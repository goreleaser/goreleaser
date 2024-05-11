package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/x/exp/ordered"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/xanzy/go-gitlab"
)

const DefaultGitLabDownloadURL = "https://gitlab.com"

var (
	_ Client            = &gitlabClient{}
	_ PullRequestOpener = &gitlabClient{}
)

type gitlabClient struct {
	client *gitlab.Client
}

// newGitLab returns a gitlab client implementation.
func newGitLab(ctx *context.Context, token string) (*gitlabClient, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			//nolint:gosec
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

	var client *gitlab.Client
	var err error
	if checkUseJobToken(*ctx, token) {
		client, err = gitlab.NewJobClient(token, options...)
	} else {
		client, err = gitlab.NewClient(token, options...)
	}
	if err != nil {
		return &gitlabClient{}, err
	}
	return &gitlabClient{client: client}, nil
}

func (c *gitlabClient) Changelog(_ *context.Context, repo Repo, prev, current string) ([]ChangelogItem, error) {
	cmpOpts := &gitlab.CompareOptions{
		From: &prev,
		To:   &current,
	}
	result, _, err := c.client.Repositories.Compare(repo.String(), cmpOpts)
	var log []ChangelogItem
	if err != nil {
		return nil, err
	}

	for _, commit := range result.Commits {
		log = append(log, ChangelogItem{
			SHA:         commit.ShortID,
			Message:     strings.Split(commit.Message, "\n")[0],
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
		})
	}
	return log, nil
}

// getDefaultBranch get the default branch
func (c *gitlabClient) getDefaultBranch(_ *context.Context, repo Repo) (string, error) {
	projectID := repo.String()
	p, res, err := c.client.Projects.GetProject(projectID, nil)
	if err != nil {
		log := log.WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).Warn("error checking for default branch")
		return "", err
	}
	return p.DefaultBranch, nil
}

// checkBranchExists checks if a branch exists
func (c *gitlabClient) checkBranchExists(_ *context.Context, repo Repo, branch string) (bool, error) {
	projectID := repo.Name
	if repo.Owner != "" {
		projectID = repo.Owner + "/" + projectID
	}

	// Verify if branch exists
	_, res, err := c.client.Branches.GetBranch(projectID, branch)
	if err != nil && res.StatusCode != 404 {
		log.WithError(err).
			Error("error verify branch existence")
		return false, err
	}

	return res.StatusCode != 404, nil
}

// CloseMilestone closes a given milestone.
func (c *gitlabClient) CloseMilestone(_ *context.Context, repo Repo, title string) error {
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
	fileName, // the path to the formula.rb
	message string, // the commit msg
) error {
	projectID := repo.Name
	if repo.Owner != "" {
		projectID = repo.Owner + "/" + projectID
	}

	log.
		WithField("projectID", projectID).
		Debug("project id")

	var branch, defaultBranch string
	var branchExists bool
	var err error
	// Use the branch if given one
	if repo.Branch != "" {
		branch = repo.Branch
		branchExists, err = c.checkBranchExists(ctx, repo, branch)
		if err != nil {
			return err
		}

		// Retrieving default branch because we need it for `start_branch`
		if !branchExists {
			defaultBranch, err = c.getDefaultBranch(ctx, repo)
			if err != nil {
				return err
			}
		}

		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			WithField("branchExists", branchExists).
			Debug("using given branch")
	} else {
		// Try to get the default branch from the Git provider
		branch, err = c.getDefaultBranch(ctx, repo)
		if err != nil {
			return err
		}

		defaultBranch = branch
		branchExists = true

		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			Debug("no branch given, using default branch")
	}

	// If the branch doesn't exist, we need to check the default branch
	// because that's what we use as `start_branch` later if the file needs
	// to be created.
	opts := &gitlab.GetFileOptions{Ref: &defaultBranch}
	if branchExists {
		opts.Ref = &branch
	}

	// Check if the file already exists
	_, res, err := c.client.RepositoryFiles.GetFile(projectID, fileName, opts)
	if err != nil && (res == nil || res.StatusCode != 404) {
		log := log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).
			Error("error getting file for brew formula")
		return err
	}

	log.
		WithField("projectID", projectID).
		WithField("branch", branch).
		WithField("fileName", fileName).
		Info("pushing file")

	stringContents := string(content)

	if res.StatusCode == 404 {
		// Create a new file because it's not already there
		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			WithField("fileName", fileName).
			Debug("file doesn't exist, creating it")

		createOpts := &gitlab.CreateFileOptions{
			AuthorName:    &commitAuthor.Name,
			AuthorEmail:   &commitAuthor.Email,
			Content:       &stringContents,
			Branch:        &branch,
			CommitMessage: &message,
		}

		// Branch not found, thus Gitlab requires a "start branch" to create the file
		if !branchExists {
			createOpts.StartBranch = &defaultBranch
		}

		fileInfo, res, err := c.client.RepositoryFiles.CreateFile(projectID, fileName, createOpts)
		if err != nil {
			log := log.
				WithField("fileName", fileName).
				WithField("branch", branch).
				WithField("projectID", projectID)
			if res != nil {
				log = log.WithField("statusCode", res.StatusCode)
			}
			log.WithError(err).
				Error("error creating brew formula file")
			return err
		}

		log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID).
			WithField("filePath", fileInfo.FilePath).
			Debug("created brew formula file")
		return nil
	}

	// Update the existing file
	log.
		WithField("fileName", fileName).
		WithField("branch", branch).
		WithField("projectID", projectID).
		Debug("file exists, updating it")

	updateOpts := &gitlab.UpdateFileOptions{
		AuthorName:    &commitAuthor.Name,
		AuthorEmail:   &commitAuthor.Email,
		Content:       &stringContents,
		Branch:        &branch,
		CommitMessage: &message,
	}

	// Branch not found, thus Gitlab requires a "start branch" to update the file
	if !branchExists {
		updateOpts.StartBranch = &defaultBranch
	}

	updateFileInfo, res, err := c.client.RepositoryFiles.UpdateFile(projectID, fileName, updateOpts)
	if err != nil {
		log := log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).
			Error("error updating file")
		return err
	}

	log := log.
		WithField("fileName", fileName).
		WithField("branch", branch).
		WithField("projectID", projectID).
		WithField("filePath", updateFileInfo.FilePath)
	if res != nil {
		log = log.WithField("statusCode", res.StatusCode)
	}
	log.Debug("updated file")
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
	log.
		WithField("owner", ctx.Config.Release.GitLab.Owner).
		WithField("name", gitlabName).
		WithField("projectID", projectID).
		Debug("projectID")

	name := title
	tagName := ctx.Git.CurrentTag
	release, resp, err := c.client.Releases.GetRelease(projectID, tagName)
	if err != nil && (resp == nil || (resp.StatusCode != 403 && resp.StatusCode != 404)) {
		return "", err
	}

	if resp.StatusCode == 403 || resp.StatusCode == 404 {
		log.WithError(err).Debug("get release")

		description := body
		ref := ctx.Git.Commit
		gitURL := ctx.Git.URL

		log.
			WithField("name", name).
			WithField("description", description).
			WithField("ref", ref).
			WithField("url", gitURL).
			Debug("creating release")
		release, _, err = c.client.Releases.CreateRelease(projectID, &gitlab.CreateReleaseOptions{
			Name:        &name,
			Description: &description,
			Ref:         &ref,
			TagName:     &tagName,
		})
		if err != nil {
			log.WithError(err).Debug("error creating release")
			return "", err
		}
		log.WithField("name", release.Name).Info("release created")
	} else {
		desc := body
		if release != nil {
			desc = getReleaseNotes(release.Description, body, ctx.Config.Release.ReleaseNotesMode)
		}

		release, _, err = c.client.Releases.UpdateRelease(projectID, tagName, &gitlab.UpdateReleaseOptions{
			Name:        &name,
			Description: &desc,
		})
		if err != nil {
			log.WithError(err).Debug("error updating release")
			return "", err
		}

		log.WithField("name", release.Name).Info("release updated")
	}

	return tagName, err // gitlab references a tag in a repo by its name
}

func (c *gitlabClient) PublishRelease(_ *context.Context, _ string /* releaseID */) (err error) {
	// GitLab doesn't support draft releases. So a created release is already published.
	return nil
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

	log.WithField("file", file.Name()).
		WithField("url", baseLinkURL).
		Debug("uploaded file")

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

	log.WithField("id", releaseLink.ID).
		WithField("url", releaseLink.DirectAssetURL).
		Debug("created release link")

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

// checkUseJobToken examines the context and given token, and determines if We should use NewJobClient vs NewClient
func checkUseJobToken(ctx context.Context, token string) bool {
	// The CI_JOB_TOKEN env var is set automatically in all GitLab runners.
	// If this comes back as empty, we aren't in a functional GitLab runner
	ciToken := os.Getenv("CI_JOB_TOKEN")
	if ciToken == "" {
		return false
	}

	// We only want to use the JobToken client if we have specified
	// UseJobToken. Older versions of GitLab don't work with this, so we
	// want to be specific
	if ctx.Config.GitLabURLs.UseJobToken {
		// We may be creating a new client with a non-CI_JOB_TOKEN, for
		// things like Homebrew publishing. We can't use the
		// CI_JOB_TOKEN there
		return token == ciToken
	}
	return false
}

func (c *gitlabClient) OpenPullRequest(
	ctx *context.Context,
	base, head Repo,
	title string,
	draft bool,
) error {
	var targetProjectID int
	if base.Owner != "" {
		fullProjectPath := fmt.Sprintf("%s/%s", base.Owner, base.Name)

		p, res, err := c.client.Projects.GetProject(fullProjectPath, nil)
		if err != nil {
			log := log.WithField("project", fullProjectPath)
			if res != nil {
				log = log.WithField("statusCode", res.StatusCode)
			}
			log.WithError(err).Warn("error getting base project id")
			return err
		}
		targetProjectID = p.ID
	}

	base.Owner = ordered.First(base.Owner, head.Owner)
	base.Name = ordered.First(base.Name, head.Name)

	if base.Branch == "" {
		def, err := c.getDefaultBranch(ctx, base)
		if err != nil {
			return err
		}
		base.Branch = def
	}

	if draft {
		title = fmt.Sprintf("Draft: %s", title)
	}

	log.WithField("base", headString(base, Repo{})).
		WithField("head", headString(base, head)).
		WithField("draft", draft).
		Info("opening pull request")

	mrOptions := &gitlab.CreateMergeRequestOptions{
		SourceBranch: &head.Branch,
		TargetBranch: &base.Branch,
		Title:        &title,
	}

	if targetProjectID != 0 {
		mrOptions.TargetProjectID = &targetProjectID
	}

	pr, _, err := c.client.MergeRequests.CreateMergeRequest(fmt.Sprintf("%s/%s", head.Owner, head.Name), mrOptions)
	if err != nil {
		return fmt.Errorf("could not create pull request: %w", err)
	}
	log.WithField("url", pr.WebURL).Info("pull request created")
	return nil
}
