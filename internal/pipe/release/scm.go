package release

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func setupGitHub(ctx *context.Context) error {
	if ctx.Config.Release.GitHub.Name == "" {
		repo, err := getRepository(ctx)
		if err != nil && !ctx.Snapshot {
			return err
		}
		ctx.Config.Release.GitHub = repo
	}

	if err := tmpl.New(ctx).ApplyAll(
		&ctx.Config.Release.GitHub.Name,
		&ctx.Config.Release.GitHub.Owner,
	); err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/releases/tag/%s",
		ctx.Config.GitHubURLs.Download,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
	))
	ctx.ReleaseURL = url
	return err
}

func setupGitLab(ctx *context.Context) error {
	if ctx.Config.Release.GitLab.Name == "" {
		repo, err := getRepository(ctx)
		if err != nil {
			return err
		}
		ctx.Config.Release.GitLab = repo
	}

	if err := tmpl.New(ctx).ApplyAll(
		&ctx.Config.Release.GitLab.Name,
		&ctx.Config.Release.GitLab.Owner,
	); err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/-/releases/%s",
		ctx.Config.GitLabURLs.Download,
		ctx.Config.Release.GitLab.Owner,
		ctx.Config.Release.GitLab.Name,
		ctx.Git.CurrentTag,
	))
	ctx.ReleaseURL = url
	return err
}

func setupGitea(ctx *context.Context) error {
	if ctx.Config.Release.Gitea.Name == "" {
		repo, err := getRepository(ctx)
		if err != nil {
			return err
		}
		ctx.Config.Release.Gitea = repo
	}

	if err := tmpl.New(ctx).ApplyAll(
		&ctx.Config.Release.Gitea.Name,
		&ctx.Config.Release.Gitea.Owner,
	); err != nil {
		return err
	}

	url, err := tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/releases/tag/%s",
		ctx.Config.GiteaURLs.Download,
		ctx.Config.Release.Gitea.Owner,
		ctx.Config.Release.Gitea.Name,
		ctx.Git.CurrentTag,
	))
	ctx.ReleaseURL = url
	return err
}
