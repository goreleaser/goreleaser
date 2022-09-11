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

	owner, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitHub.Owner)
	if err != nil {
		return err
	}
	ctx.Config.Release.GitHub.Owner = owner

	name, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitHub.Name)
	if err != nil {
		return err
	}
	ctx.Config.Release.GitHub.Name = name

	ctx.ReleaseURL, err = tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/releases/tag/%s",
		ctx.Config.GitHubURLs.Download,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
	))
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

	owner, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Owner)
	if err != nil {
		return err
	}
	ctx.Config.Release.GitLab.Owner = owner

	name, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return err
	}
	ctx.Config.Release.GitLab.Name = name

	ctx.ReleaseURL, err = tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/-/releases/%s",
		ctx.Config.GitLabURLs.Download,
		ctx.Config.Release.GitLab.Owner,
		ctx.Config.Release.GitLab.Name,
		ctx.Git.CurrentTag,
	))
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

	owner, err := tmpl.New(ctx).Apply(ctx.Config.Release.Gitea.Owner)
	if err != nil {
		return err
	}
	ctx.Config.Release.Gitea.Owner = owner

	name, err := tmpl.New(ctx).Apply(ctx.Config.Release.Gitea.Name)
	if err != nil {
		return err
	}
	ctx.Config.Release.Gitea.Name = name

	ctx.ReleaseURL, err = tmpl.New(ctx).Apply(fmt.Sprintf(
		"%s/%s/%s/releases/tag/%s",
		ctx.Config.GiteaURLs.Download,
		ctx.Config.Release.Gitea.Owner,
		ctx.Config.Release.Gitea.Name,
		ctx.Git.CurrentTag,
	))
	return err
}
