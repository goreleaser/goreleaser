package client

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/apex/log"
	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/oauth2"
)

type githubClient struct {
	client *github.Client
}

// NewGitHub returns a github client implementation
func NewGitHub(ctx *context.Context) (Client, error) {
	if ctx.Config.RepoURLs.API == "" {
		ctx.Config.RepoURLs.API = "https://api.github.com"
	}
	if ctx.Config.RepoURLs.Download == "" {
		ctx.Config.RepoURLs.Download = "https://github.com"
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ctx.StorageToken},
	)
	client := github.NewClient(oauth2.NewClient(ctx, ts))
	api, err := url.Parse(ctx.Config.RepoURLs.API)
	if err != nil {
		return &githubClient{}, err
	}
	upload, err := url.Parse(ctx.Config.RepoURLs.Upload)
	if err != nil {
		return &githubClient{}, err
	}
	client.BaseURL = api
	client.UploadURL = upload

	return &githubClient{client: client}, nil
}

func (c *githubClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo config.Repo,
	content bytes.Buffer,
	path string,
	message string,
) error {
	options := &github.RepositoryContentFileOptions{
		Committer: &github.CommitAuthor{
			Name:  github.String(commitAuthor.Name),
			Email: github.String(commitAuthor.Email),
		},
		Content: content.Bytes(),
		Message: github.String(message),
	}

	file, _, res, err := c.client.Repositories.GetContents(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		&github.RepositoryContentGetOptions{},
	)
	if err != nil && res.StatusCode != 404 {
		return err
	}

	if res.StatusCode == 404 {
		_, _, err = c.client.Repositories.CreateFile(
			ctx,
			repo.Owner,
			repo.Name,
			path,
			options,
		)
		return err
	}
	options.SHA = file.SHA
	_, _, err = c.client.Repositories.UpdateFile(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		options,
	)
	return err
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	var release *github.RepositoryRelease
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
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
		ctx.Config.Release.Repo.Owner,
		ctx.Config.Release.Repo.Name,
		ctx.Git.CurrentTag,
	)
	if err != nil {
		release, _, err = c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.Repo.Owner,
			ctx.Config.Release.Repo.Name,
			data,
		)
	} else {
		release, _, err = c.client.Repositories.EditRelease(
			ctx,
			ctx.Config.Release.Repo.Owner,
			ctx.Config.Release.Repo.Name,
			release.GetID(),
			data,
		)
	}
	log.WithField("url", release.GetHTMLURL()).Info("release updated")
	return strconv.Itoa(int(release.GetID())), err
}

func (c *githubClient) Upload(
	ctx *context.Context,
	releaseID string,
	name string,
	file *os.File,
) (string, error) {

	release, err := strconv.ParseInt(releaseID, 10, 0)
	if err != nil {
		return "", err
	}

	_, _, err = c.client.Repositories.UploadReleaseAsset(
		ctx,
		ctx.Config.Release.Repo.Owner,
		ctx.Config.Release.Repo.Name,
		release,
		&github.UploadOptions{
			Name: name,
		},
		file,
	)

	path := fmt.Sprintf(
		"/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
		ctx.Config.Release.Repo.Owner,
		ctx.Config.Release.Repo.Name,
	)

	return path, err
}
