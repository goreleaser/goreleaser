package client

import (
	"bytes"
	"fmt"
	"log"
	"os"

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
	var title = fmt.Sprintf(
		"Releasing %v version %v",
		ctx.Config.Build.Binary,
		ctx.Git.CurrentTag,
	)
	master, branch, err := c.setupBranches(ctx)
	if err != nil {
		return err
	}
	if err = c.uploadFile(ctx, content, path, branch, title); err != nil {
		return err
	}
	pull, _, err := c.client.PullRequests.Create(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		&github.NewPullRequest{
			Base:  master.Ref,
			Head:  branch.Ref,
			Title: github.String(title),
		},
	)
	if err != nil {
		return err
	}
	log.Printf("Created pull request %v", pull.GetHTMLURL())
	if ctx.Config.Release.Draft {
		log.Println("Draft release, pull request will not be merged.")
		return nil
	}
	return c.mergeAndDeleteBranch(ctx, pull, branch)
}

func (c *githubClient) setupBranches(
	ctx *context.Context,
) (master, branch *github.Reference, err error) {
	master, _, err = c.client.Git.GetRef(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		"refs/heads/master",
	)
	if err != nil {
		return
	}
	branch, _, err = c.client.Git.CreateRef(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		&github.Reference{
			Ref: github.String(fmt.Sprintf(
				"refs/heads/%v-%v",
				ctx.Config.Build.Binary,
				ctx.Git.CurrentTag,
			)),
			Object: &github.GitObject{
				SHA: github.String(master.Object.GetSHA()),
			},
		},
	)
	return
}

func (c *githubClient) uploadFile(
	ctx *context.Context,
	content bytes.Buffer,
	path string,
	branch *github.Reference,
	msg string,
) error {
	file, _, res, err := c.client.Repositories.GetContents(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		path,
		&github.RepositoryContentGetOptions{},
	)
	var options = &github.RepositoryContentFileOptions{
		Committer: &github.CommitAuthor{
			Name:  github.String("goreleaser"),
			Login: github.String("goreleaser"),
			Email: github.String("goreleaser@goreleaser"),
		},
		Content: content.Bytes(),
		Message: github.String(msg),
		Branch:  branch.Ref,
	}
	if err != nil && res.StatusCode == 404 {
		if _, _, err = c.client.Repositories.CreateFile(
			ctx,
			ctx.Config.Brew.GitHub.Owner,
			ctx.Config.Brew.GitHub.Name,
			path,
			options,
		); err != nil {
			return err
		}
	}
	options.SHA = file.SHA
	_, _, err = c.client.Repositories.UpdateFile(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		path,
		options,
	)
	return err
}

func (c *githubClient) mergeAndDeleteBranch(
	ctx *context.Context,
	pull *github.PullRequest,
	branch *github.Reference,
) error {
	if _, _, err := c.client.PullRequests.Merge(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		pull.GetNumber(),
		pull.GetTitle(),
		&github.PullRequestOptions{},
	); err != nil {
		return err
	}
	_, err := c.client.Git.DeleteRef(
		ctx,
		ctx.Config.Brew.GitHub.Owner,
		ctx.Config.Brew.GitHub.Name,
		branch.GetRef(),
	)
	return err
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (releaseID int, err error) {
	var data = &github.RepositoryRelease{
		Name:    github.String(ctx.Git.CurrentTag),
		TagName: github.String(ctx.Git.CurrentTag),
		Body:    github.String(body),
		Draft:   github.Bool(ctx.Config.Release.Draft),
	}
	r, _, err := c.client.Repositories.GetReleaseByTag(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
	)
	if err != nil {
		r, _, err = c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			data,
		)
		log.Printf("URL: %v\n", r.GetHTMLURL())
		return r.GetID(), err
	}
	r, _, err = c.client.Repositories.EditRelease(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		r.GetID(),
		data,
	)
	log.Printf("URL: %v\n", r.GetHTMLURL())
	return r.GetID(), err
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
