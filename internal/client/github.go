package client

import (
	"bytes"
	"os"

	"github.com/apex/log"
	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/oauth2"
)

type githubClient struct {
	client *github.Client
}

// NewGitHub returns a github client implementation
func NewGitHub(ctx *context.Context) Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ctx.Token},
	)
	return &githubClient{
		client: github.NewClient(oauth2.NewClient(ctx, ts)),
	}
}

func (c *githubClient) CreateFile(
	ctx *context.Context,
	content bytes.Buffer,
	path string,
) (err error) {
	options := &github.RepositoryContentFileOptions{
		Committer: &github.CommitAuthor{
			Name:  github.String("goreleaserbot"),
			Email: github.String("goreleaser@carlosbecker.com"),
		},
		Content: content.Bytes(),
		Message: github.String(
			ctx.Config.ProjectName + " version " + ctx.Git.CurrentTag,
		),
	}

	file, _, res, err := c.client.Repositories.GetContents(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		path,
		&github.RepositoryContentGetOptions{},
	)
	if err != nil && res.StatusCode == 404 {
		_, _, err = c.client.Repositories.CreateFile(
			ctx,
			ctx.Config.Brew.GitHub.Owner,
			ctx.Config.Brew.GitHub.Name,
			path,
			options,
		)
		return
	}
	options.SHA = file.SHA
	_, _, err = c.client.Repositories.UpdateFile(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		path,
		options,
	)
	return
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (releaseID int, err error) {
	var release *github.RepositoryRelease
	var data = &github.RepositoryRelease{
		Name:    github.String(ctx.Git.CurrentTag),
		TagName: github.String(ctx.Git.CurrentTag),
		Body:    github.String(body),
		Draft:   github.Bool(ctx.Config.Release.Draft),
	}
	release, _, err = c.client.Repositories.GetReleaseByTag(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
	)
	if err != nil {
		release, _, err = c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			data,
		)
	} else {
		release, _, err = c.client.Repositories.EditRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			release.GetID(),
			data,
		)
	}
	log.WithField("url", release.GetHTMLURL()).Info("release updated")
	return release.GetID(), err
}

func (c *githubClient) Upload(
	ctx *context.Context,
	releaseID int,
	name string,
	file *os.File,
) (err error) {
	_, _, err = c.client.Repositories.UploadReleaseAsset(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		releaseID,
		&github.UploadOptions{
			Name: name,
		},
		file,
	)
	return
}
