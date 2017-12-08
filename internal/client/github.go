package client

import (
	"bytes"
	"net/url"
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
func NewGitHub(ctx *context.Context) (Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ctx.Token},
	)
	client := github.NewClient(oauth2.NewClient(ctx, ts))
	if ctx.Config.GitHubURLs.API != "" {
		api, err := url.Parse(ctx.Config.GitHubURLs.API)
		if err != nil {
			return &githubClient{}, err
		}
		upload, err := url.Parse(ctx.Config.GitHubURLs.Upload)
		if err != nil {
			return &githubClient{}, err
		}
		client.BaseURL = api
		client.UploadURL = upload
	}

	return &githubClient{client}, nil
}

func (c *githubClient) CreateFile(
	ctx *context.Context,
	content bytes.Buffer,
	path string,
) (err error) {
	options := &github.RepositoryContentFileOptions{
		Committer: &github.CommitAuthor{
			Name:  github.String(ctx.Config.Brew.CommitAuthor.Name),
			Email: github.String(ctx.Config.Brew.CommitAuthor.Email),
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
	title, err := releaseTitle(ctx)
	if err != nil {
		return 0, err
	}
	var data = &github.RepositoryRelease{
		Name:       github.String(title),
		TagName:    github.String(ctx.Git.CurrentTag),
		Body:       github.String(body),
		Draft:      github.Bool(ctx.Config.Release.Draft),
		Prerelease: github.Bool(ctx.Config.Release.Prerelease),
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
