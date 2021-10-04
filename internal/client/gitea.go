package client

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"code.gitea.io/sdk/gitea"
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type giteaClient struct {
	client *gitea.Client
}

func getInstanceURL(ctx *context.Context) (string, error) {
	apiURL, err := tmpl.New(ctx).Apply(ctx.Config.GiteaURLs.API)
	if err != nil {
		return "", fmt.Errorf("templating Gitea API URL: %w", err)
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	u.Path = ""
	rawurl := u.String()
	if rawurl == "" {
		return "", fmt.Errorf("invalid URL: %v", apiURL)
	}
	return rawurl, nil
}

// NewGitea returns a gitea client implementation.
func NewGitea(ctx *context.Context, token string) (Client, error) {
	instanceURL, err := getInstanceURL(ctx)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			// nolint: gosec
			InsecureSkipVerify: ctx.Config.GiteaURLs.SkipTLSVerify,
		},
	}
	httpClient := &http.Client{Transport: transport}
	client, err := gitea.NewClient(instanceURL,
		gitea.SetToken(token),
		gitea.SetHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		gitea.SetContext(ctx)(client)
	}
	return &giteaClient{client: client}, nil
}

func (c *giteaClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	return "", ErrNotImplemented
}

// CloseMilestone closes a given milestone.
func (c *giteaClient) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	closedState := gitea.StateClosed
	opts := gitea.EditMilestoneOption{
		State: &closedState,
		Title: title,
	}

	_, resp, err := c.client.EditMilestoneByName(repo.Owner, repo.Name, title, opts)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNoMilestoneFound{Title: title}
	}
	return err
}

func (c *giteaClient) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	projectID := repo.String()
	p, res, err := c.client.GetRepo(repo.Owner, repo.Name)
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

// CreateFile creates a file in the repository at a given path
// or updates the file if it exists.
func (c *giteaClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	content []byte,
	path,
	message string,
) error {
	// use default branch
	var branch string
	var err error
	if repo.Branch != "" {
		branch = repo.Branch
	} else {
		branch, err = c.GetDefaultBranch(ctx, repo)
		if err != nil {
			// Fall back to 'master' 😭
			log.WithFields(log.Fields{
				"fileName":        path,
				"projectID":       repo.String(),
				"requestedBranch": branch,
				"err":             err.Error(),
			}).Warn("error checking for default branch, using master")
		}

	}

	fileOptions := gitea.FileOptions{
		Message:    message,
		BranchName: branch,
		Author: gitea.Identity{
			Name:  commitAuthor.Name,
			Email: commitAuthor.Email,
		},
		Committer: gitea.Identity{
			Name:  commitAuthor.Name,
			Email: commitAuthor.Email,
		},
	}

	currentFile, resp, err := c.client.GetContents(repo.Owner, repo.Name, branch, path)
	// file not exist, create it
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
		_, _, err = c.client.CreateFile(repo.Owner, repo.Name, path, gitea.CreateFileOptions{
			FileOptions: fileOptions,
			Content:     base64.StdEncoding.EncodeToString(content),
		})
		return err
	}

	// update file
	_, _, err = c.client.UpdateFile(repo.Owner, repo.Name, path, gitea.UpdateFileOptions{
		FileOptions: fileOptions,
		SHA:         currentFile.SHA,
		Content:     base64.StdEncoding.EncodeToString(content),
	})
	return err
}

func (c *giteaClient) createRelease(ctx *context.Context, title, body string) (*gitea.Release, error) {
	releaseConfig := ctx.Config.Release
	owner := releaseConfig.Gitea.Owner
	repoName := releaseConfig.Gitea.Name
	tag := ctx.Git.CurrentTag

	opts := gitea.CreateReleaseOption{
		TagName:      tag,
		Target:       ctx.Git.Commit,
		Title:        title,
		Note:         body,
		IsDraft:      releaseConfig.Draft,
		IsPrerelease: ctx.PreRelease,
	}
	release, _, err := c.client.CreateRelease(owner, repoName, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Debug("error creating Gitea release")
		return nil, err
	}
	log.WithField("id", release.ID).Info("Gitea release created")
	return release, nil
}

func (c *giteaClient) getExistingRelease(owner, repoName, tagName string) (*gitea.Release, error) {
	releases, _, err := c.client.ListReleases(owner, repoName, gitea.ListReleasesOptions{})
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.TagName == tagName {
			return release, nil
		}
	}

	return nil, nil
}

func (c *giteaClient) updateRelease(ctx *context.Context, title, body string, id int64) (*gitea.Release, error) {
	releaseConfig := ctx.Config.Release
	owner := releaseConfig.Gitea.Owner
	repoName := releaseConfig.Gitea.Name
	tag := ctx.Git.CurrentTag

	opts := gitea.EditReleaseOption{
		TagName:      tag,
		Target:       ctx.Git.Commit,
		Title:        title,
		Note:         body,
		IsDraft:      &releaseConfig.Draft,
		IsPrerelease: &ctx.PreRelease,
	}

	release, _, err := c.client.EditRelease(owner, repoName, id, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err.Error(),
		}).Debug("error updating Gitea release")
		return nil, err
	}
	log.WithField("id", release.ID).Info("Gitea release updated")
	return release, nil
}

// CreateRelease creates a new release or updates it by keeping
// the release notes if it exists.
func (c *giteaClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	var release *gitea.Release
	var err error

	releaseConfig := ctx.Config.Release

	title, err := tmpl.New(ctx).Apply(releaseConfig.NameTemplate)
	if err != nil {
		return "", err
	}

	release, err = c.getExistingRelease(
		releaseConfig.Gitea.Owner,
		releaseConfig.Gitea.Name,
		ctx.Git.CurrentTag,
	)
	if err != nil {
		return "", err
	}

	if release != nil {
		if release.Note != "" {
			body = release.Note
		}
		release, err = c.updateRelease(ctx, title, body, release.ID)
		if err != nil {
			return "", err
		}
	} else {
		release, err = c.createRelease(ctx, title, body)
		if err != nil {
			return "", err
		}
	}

	return strconv.FormatInt(release.ID, 10), nil
}

func (c *giteaClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	downloadURL, err := tmpl.New(ctx).Apply(ctx.Config.GiteaURLs.Download)
	if err != nil {
		return "", fmt.Errorf("templating Gitea download URL: %w", err)
	}

	return fmt.Sprintf(
		"%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
		downloadURL,
		ctx.Config.Release.Gitea.Owner,
		ctx.Config.Release.Gitea.Name,
	), nil
}

// Upload uploads a file into a release repository.
func (c *giteaClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
	file *os.File,
) error {
	giteaReleaseID, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return err
	}
	releaseConfig := ctx.Config.Release
	owner := releaseConfig.Gitea.Owner
	repoName := releaseConfig.Gitea.Name

	_, _, err = c.client.CreateReleaseAttachment(owner, repoName, giteaReleaseID, file, artifact.Name)
	if err != nil {
		return RetriableError{err}
	}
	return nil
}
