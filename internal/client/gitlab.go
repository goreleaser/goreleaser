package client

import (
	"bytes"
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/xanzy/go-gitlab"
)

type gitlabClient struct {
	client *gitlab.Client
}

// NewGitLab returns a gitlab client implementation
func NewGitLab(ctx *context.Context) (Client, error) {
	token := ctx.Token
	client := gitlab.NewClient(nil, token)
	if ctx.Config.GitLabURLs.API != "" {
		err := client.SetBaseURL(ctx.Config.GitLabURLs.API)
		if err != nil {
			return &gitlabClient{}, err
		}
	}
	return &gitlabClient{client: client}, nil
}

// CreateFile creates a file in the repository at a given path
// or updates the file if it exists
func (c *gitlabClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo config.Repo,
	content bytes.Buffer,
	path,
	message string,
) error {
	// TODO how to get already uploaded files/content for a project?
	// https://docs.gitlab.com/ce/api/projects.html#upload-a-file
	// for now we always upload the same one but gitlab genereates
	// a new hash each time
	projectID := repo.Owner + "/" + repo.Name
	_, _, err := c.client.Projects.UploadFile(
		projectID,
		path,
		nil,
	)
	return err
}

// CreateRelease creates a new release or updates it by keeping
// the release notes if it exists
func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}

	projectID := ctx.Config.Release.GitHub.Owner + "/" + ctx.Config.Release.GitHub.Owner
	name := title
	tagName := ctx.Git.CurrentTag
	release, resp, err := c.client.Releases.GetRelease(projectID, tagName)
	if err != nil && resp.StatusCode == 403 {
		desc := body
		ref := ctx.Git.Commit
		release, _, err = c.client.Releases.CreateRelease(projectID, &gitlab.CreateReleaseOptions{
			Name:        &name,
			Description: &desc,
			Ref:         &ref,
			TagName:     &tagName,
		})
		log.WithField("name", release.Name).Info("release created")
	} else {
		desc := body
		if release.DescriptionHTML != "" {
			desc = release.DescriptionHTML
		}

		release, _, err = c.client.Releases.UpdateRelease(projectID, tagName, &gitlab.UpdateReleaseOptions{
			Name:        &name,
			Description: &desc,
		})
		log.WithField("name", release.Name).Info("release updated")
	}

	return tagName, err // gitlab references a tag in a repo by its name
}

// Upload uploads a file into a release repository
func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID string,
	name string,
	file *os.File,
) error {
	gitlabBaseURL := ctx.Config.GitLabURLs.API
	projectID := ctx.Config.Release.GitHub.Owner + "/" + ctx.Config.Release.GitHub.Owner
	// projectFile from upload: /uploads/<sha>/filename.txt
	relativeUploadURL := "TODO" // from context
	linkURL := gitlabBaseURL + "/" + projectID + relativeUploadURL
	_, _, err := c.client.ReleaseLinks.CreateReleaseLink(
		projectID,
		releaseID,
		&gitlab.CreateReleaseLinkOptions{
			Name: &name,
			URL:  &linkURL,
		})
	return err
}
