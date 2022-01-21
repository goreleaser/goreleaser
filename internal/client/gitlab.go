package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/xanzy/go-gitlab"
)

const DefaultGitLabDownloadURL = "https://gitlab.com"

type gitlabClient struct {
	client *gitlab.Client
}

// NewGitLab returns a gitlab client implementation.
func NewGitLab(ctx *context.Context, token string) (Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			// nolint: gosec
			InsecureSkipVerify: ctx.Config.GitLabURLs.SkipTLSVerify,
		},
	}
	options := []gitlab.ClientOptionFunc{
		gitlab.WithHTTPClient(&http.Client{
			Transport: transport,
		}),
	}
	if ctx.Config.GitLabURLs.API != "" {
		apiURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.API)
		if err != nil {
			return nil, fmt.Errorf("templating GitLab API URL: %w", err)
		}

		options = append(options, gitlab.WithBaseURL(apiURL))
	}
	client, err := gitlab.NewClient(token, options...)
	if err != nil {
		return &gitlabClient{}, err
	}
	return &gitlabClient{client: client}, nil
}

func (c *gitlabClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	cmpOpts := &gitlab.CompareOptions{
		From: &prev,
		To:   &current,
	}
	result, _, err := c.client.Repositories.Compare(repo.String(), cmpOpts)
	var log []string
	if err != nil {
		return "", err
	}

	for _, commit := range result.Commits {
		log = append(log, fmt.Sprintf(
			"%s: %s (%s <%s>)",
			commit.ShortID,
			strings.Split(commit.Message, "\n")[0],
			commit.AuthorName,
			commit.AuthorEmail,
		))
	}
	return strings.Join(log, "\n"), nil
}

// GetDefaultBranch get the default branch
func (c *gitlabClient) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	projectID := repo.String()
	p, res, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID":  projectID,
			"statusCode": res.StatusCode,
			"err":        err.Error(),
		}).Warn("error checking for default branch")
		return "", err
	}
	return p.DefaultBranch, nil
}

// CloseMilestone closes a given milestone.
func (c *gitlabClient) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	milestone, err := c.getMilestoneByTitle(repo, title)
	if err != nil {
		return err
	}

	if milestone == nil {
		return ErrNoMilestoneFound{Title: title}
	}

	closeStateEvent := "close"

	opts := &gitlab.UpdateMilestoneOptions{
		Description: &milestone.Description,
		DueDate:     milestone.DueDate,
		StartDate:   milestone.StartDate,
		StateEvent:  &closeStateEvent,
		Title:       &milestone.Title,
	}

	_, _, err = c.client.Milestones.UpdateMilestone(
		repo.String(),
		milestone.ID,
		opts,
	)

	return err
}

// CreateFile gets a file in the repository at a given path
// and updates if it exists or creates it for later pipes in the pipeline.
func (c *gitlabClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	content []byte, // the content of the formula.rb
	path, // the path to the formula.rb
	message string, // the commit msg
) error {
	fileName := path
	projectID := repo.String()

	// Use the project default branch if we can get it...otherwise, just use
	// 'master'
	var branch, ref string
	var err error
	// Use the branch if given one
	if repo.Branch != "" {
		branch = repo.Branch
	} else {
		// Try to get the default branch from the Git provider
		branch, err = c.GetDefaultBranch(ctx, repo)
		if err != nil {
			// Fall back to 'master' 😭
			log.WithFields(log.Fields{
				"fileName":        fileName,
				"projectID":       repo.String(),
				"requestedBranch": branch,
				"err":             err.Error(),
			}).Warn("error checking for default branch, using master")
			ref = "master"
			branch = "master"
		}
	}
	ref = branch
	opts := &gitlab.GetFileOptions{Ref: &ref}
	castedContent := string(content)

	log.WithFields(log.Fields{
		"owner":  repo.Owner,
		"name":   repo.Name,
		"ref":    ref,
		"branch": branch,
	}).Debug("projectID at brew")

	_, res, err := c.client.RepositoryFiles.GetFile(repo.String(), fileName, opts)
	if err != nil && (res == nil || res.StatusCode != 404) {
		log.WithFields(log.Fields{
			"fileName":   fileName,
			"ref":        ref,
			"projectID":  projectID,
			"statusCode": res.StatusCode,
			"err":        err.Error(),
		}).Error("error getting file for brew formula")
		return err
	}

	log.WithFields(log.Fields{
		"fileName":  fileName,
		"branch":    branch,
		"projectID": projectID,
	}).Debug("found already existing brew formula file")

	if res.StatusCode == 404 {
		log.WithFields(log.Fields{
			"fileName":  fileName,
			"ref":       ref,
			"projectID": projectID,
		}).Debug("creating brew formula")
		createOpts := &gitlab.CreateFileOptions{
			AuthorName:    &commitAuthor.Name,
			AuthorEmail:   &commitAuthor.Email,
			Content:       &castedContent,
			Branch:        &branch,
			CommitMessage: &message,
		}
		fileInfo, res, err := c.client.RepositoryFiles.CreateFile(projectID, fileName, createOpts)
		if err != nil {
			log.WithFields(log.Fields{
				"fileName":   fileName,
				"branch":     branch,
				"projectID":  projectID,
				"statusCode": res.StatusCode,
				"err":        err.Error(),
			}).Error("error creating brew formula file")
			return err
		}

		log.WithFields(log.Fields{
			"fileName":  fileName,
			"branch":    branch,
			"projectID": projectID,
			"filePath":  fileInfo.FilePath,
		}).Debug("created brew formula file")
		return nil
	}

	log.WithFields(log.Fields{
		"fileName":  fileName,
		"ref":       ref,
		"projectID": projectID,
	}).Debug("updating brew formula")
	updateOpts := &gitlab.UpdateFileOptions{
		AuthorName:    &commitAuthor.Name,
		AuthorEmail:   &commitAuthor.Email,
		Content:       &castedContent,
		Branch:        &branch,
		CommitMessage: &message,
	}

	updateFileInfo, res, err := c.client.RepositoryFiles.UpdateFile(projectID, fileName, updateOpts)
	if err != nil {
		log.WithFields(log.Fields{
			"fileName":   fileName,
			"branch":     branch,
			"projectID":  projectID,
			"statusCode": res.StatusCode,
			"err":        err.Error(),
		}).Error("error updating brew formula file")
		return err
	}

	log.WithFields(log.Fields{
		"fileName":   fileName,
		"branch":     branch,
		"projectID":  projectID,
		"filePath":   updateFileInfo.FilePath,
		"statusCode": res.StatusCode,
	}).Debug("updated brew formula file")
	return nil
}

// CreateRelease creates a new release or updates it by keeping
// the release notes if it exists.
func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return "", err
	}
	projectID := gitlabName
	if ctx.Config.Release.GitLab.Owner != "" {
		projectID = ctx.Config.Release.GitLab.Owner + "/" + projectID
	}
	log.WithFields(log.Fields{
		"owner":     ctx.Config.Release.GitLab.Owner,
		"name":      gitlabName,
		"projectID": projectID,
	}).Debug("projectID")

	name := title
	tagName := ctx.Git.CurrentTag
	release, resp, err := c.client.Releases.GetRelease(projectID, tagName)
	if err != nil && (resp == nil || (resp.StatusCode != 403 && resp.StatusCode != 404)) {
		return "", err
	}

	if resp.StatusCode == 403 || resp.StatusCode == 404 {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Debug("get release")

		description := body
		ref := ctx.Git.Commit
		gitURL := ctx.Git.URL

		log.WithFields(log.Fields{
			"name":        name,
			"description": description,
			"ref":         ref,
			"url":         gitURL,
		}).Debug("creating release")
		release, _, err = c.client.Releases.CreateRelease(projectID, &gitlab.CreateReleaseOptions{
			Name:        &name,
			Description: &description,
			Ref:         &ref,
			TagName:     &tagName,
		})

		if err != nil {
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Debug("error create release")
			return "", err
		}
		log.WithField("name", release.Name).Info("release created")
	} else {
		desc := body
		if release != nil {
			desc = getReleaseNotes(release.DescriptionHTML, body, ctx.Config.Release.ReleaseNotesMode)
		}

		release, _, err = c.client.Releases.UpdateRelease(projectID, tagName, &gitlab.UpdateReleaseOptions{
			Name:        &name,
			Description: &desc,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"err": err.Error(),
			}).Debug("error update release")
			return "", err
		}

		log.WithField("name", release.Name).Info("release updated")
	}

	return tagName, err // gitlab references a tag in a repo by its name
}

func (c *gitlabClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	var urlTemplate string
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return "", err
	}
	downloadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.Download)
	if err != nil {
		return "", err
	}

	if ctx.Config.Release.GitLab.Owner != "" {
		urlTemplate = fmt.Sprintf(
			"%s/%s/%s/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
			downloadURL,
			ctx.Config.Release.GitLab.Owner,
			gitlabName,
		)
	} else {
		urlTemplate = fmt.Sprintf(
			"%s/%s/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
			downloadURL,
			gitlabName,
		)
	}
	return urlTemplate, nil
}

// Upload uploads a file into a release repository.
func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
	file *os.File,
) error {
	// create new template and apply name field
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return err
	}
	projectID := gitlabName
	// check if owner is empty
	if ctx.Config.Release.GitLab.Owner != "" {
		projectID = ctx.Config.Release.GitLab.Owner + "/" + projectID
	}

	var baseLinkURL string
	var linkURL string
	if ctx.Config.GitLabURLs.UsePackageRegistry {
		log.WithField("file", file.Name()).Debug("uploading file as generic package")
		if _, _, err := c.client.GenericPackages.PublishPackageFile(
			projectID,
			ctx.Config.ProjectName,
			ctx.Version,
			artifact.Name,
			file,
			nil,
		); err != nil {
			return err
		}

		baseLinkURL, err = c.client.GenericPackages.FormatPackageURL(
			projectID,
			ctx.Config.ProjectName,
			ctx.Version,
			artifact.Name,
		)
		if err != nil {
			return err
		}
		linkURL = c.client.BaseURL().String() + baseLinkURL
	} else {
		log.WithField("file", file.Name()).Debug("uploading file as attachment")
		projectFile, _, err := c.client.Projects.UploadFile(
			projectID,
			file,
			filepath.Base(file.Name()),
			nil,
		)
		if err != nil {
			return err
		}

		baseLinkURL = projectFile.URL
		gitlabBaseURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.Download)
		if err != nil {
			return fmt.Errorf("templating GitLab Download URL: %w", err)
		}

		// search for project details based on projectID
		projectDetails, _, err := c.client.Projects.GetProject(projectID, nil)
		if err != nil {
			return err
		}
		linkURL = gitlabBaseURL + "/" + projectDetails.PathWithNamespace + baseLinkURL
	}

	log.WithFields(log.Fields{
		"file": file.Name(),
		"url":  baseLinkURL,
	}).Debug("uploaded file")

	name := artifact.Name
	filename := "/" + name
	releaseLink, _, err := c.client.ReleaseLinks.CreateReleaseLink(
		projectID,
		releaseID,
		&gitlab.CreateReleaseLinkOptions{
			Name:     &name,
			URL:      &linkURL,
			FilePath: &filename,
		})
	if err != nil {
		return RetriableError{err}
	}

	log.WithFields(log.Fields{
		"id":  releaseLink.ID,
		"url": releaseLink.DirectAssetURL,
	}).Debug("created release link")

	// for checksums.txt the field is nil, so we initialize it
	if artifact.Extra == nil {
		artifact.Extra = make(map[string]interface{})
	}

	return nil
}

// getMilestoneByTitle returns a milestone by title.
func (c *gitlabClient) getMilestoneByTitle(repo Repo, title string) (*gitlab.Milestone, error) {
	opts := &gitlab.ListMilestonesOptions{
		Title: &title,
	}

	for {
		milestones, resp, err := c.client.Milestones.ListMilestones(repo.String(), opts)
		if err != nil {
			return nil, err
		}

		for _, milestone := range milestones {
			if milestone != nil && milestone.Title == title {
				return milestone, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return nil, nil
}
