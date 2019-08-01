package client

import (
	"crypto/tls"
	"net/http"
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
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			// nolint: gosec
			InsecureSkipVerify: ctx.Config.GitLabURLs.SkipTLSVerify,
		},
	}
	httpClient := &http.Client{Transport: transport}
	client := gitlab.NewClient(httpClient, token)
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
	content []byte,
	path,
	message string,
) error {
	// Used by brew and scoop, hence those two pipes are
	// only supported for github atm. So we disable it for now.
	return nil
}

// CreateRelease creates a new release or updates it by keeping
// the release notes if it exists
func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}

	projectID := ctx.Config.Release.GitLab.Owner + "/" + ctx.Config.Release.GitLab.Name
	log.WithFields(log.Fields{
		"owner": ctx.Config.Release.GitLab.Owner,
		"name":  ctx.Config.Release.GitLab.Name,
	}).Debug("projectID")

	name := title
	tagName := ctx.Git.CurrentTag
	release, resp, err := c.client.Releases.GetRelease(projectID, tagName)
	if err != nil && (resp == nil || resp.StatusCode != 403) {
		return "", err
	}

	if resp.StatusCode == 403 {
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
		if release != nil && release.DescriptionHTML != "" {
			desc = release.DescriptionHTML
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

// Upload uploads a file into a release repository
func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID string,
	name string,
	file *os.File,
) error {
	projectID := ctx.Config.Release.GitLab.Owner + "/" + ctx.Config.Release.GitLab.Name

	log.WithField("file", file.Name()).Debug("uploading file")
	projectFile, _, err := c.client.Projects.UploadFile(
		projectID,
		file.Name(),
		nil,
	)

	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"file": file.Name(),
		"url":  projectFile.URL,
	}).Debug("uploaded file")

	gitlabBaseURL := ctx.Config.GitLabURLs.Download
	// projectFile from upload: /uploads/<sha>/filename.txt
	relativeUploadURL := projectFile.URL
	linkURL := gitlabBaseURL + "/" + projectID + relativeUploadURL
	releaseLink, _, err := c.client.ReleaseLinks.CreateReleaseLink(
		projectID,
		releaseID,
		&gitlab.CreateReleaseLinkOptions{
			Name: &name,
			URL:  &linkURL,
		})

	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"id":  releaseLink.ID,
		"url": releaseLink.URL,
	}).Debug("created release link")

	return err
}
