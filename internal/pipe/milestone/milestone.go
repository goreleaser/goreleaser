package milestone

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const defaultNameTemplate = "{{ .Tag }}"

// Pipe for milestone.
type Pipe struct{}

func (Pipe) String() string {
	return "milestones"
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if len(ctx.Config.Milestones) == 0 {
		ctx.Config.Milestones = append(ctx.Config.Milestones, config.Milestone{})
	}

	for i := range ctx.Config.Milestones {
		milestone := &ctx.Config.Milestones[i]

		if milestone.NameTemplate == "" {
			milestone.NameTemplate = defaultNameTemplate
		}

		// nolint: exhaustive
		switch ctx.TokenType {
		case context.TokenTypeGitLab:
			{
				if milestone.GitLab.Name == "" {
					repo, err := git.ExtractRepoFromConfig()
					if err != nil {
						return err
					}
					milestone.GitLab = repo
				}

				return nil
			}
		case context.TokenTypeGitea:
			{
				if milestone.Gitea.Name == "" {
					repo, err := git.ExtractRepoFromConfig()
					if err != nil {
						return err
					}
					milestone.Gitea = repo
				}

				return nil
			}
		}

		if milestone.GitHub.Name == "" {
			repo, err := git.ExtractRepoFromConfig()
			if err != nil && !ctx.Snapshot {
				return err
			}
			milestone.GitHub = repo
		}
	}

	return nil
}

// Publish the release.
func (Pipe) Publish(ctx *context.Context) error {
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	c, err := client.New(ctx)
	if err != nil {
		return err
	}
	return doPublish(ctx, c)
}

func doPublish(ctx *context.Context, vcsClient client.Client) error {
	for i := range ctx.Config.Milestones {
		milestone := &ctx.Config.Milestones[i]

		if !milestone.Close {
			return pipe.Skip("milestone pipe is disabled")
		}

		name, err := tmpl.New(ctx).Apply(milestone.NameTemplate)

		if err != nil {
			return err
		}

		var repo client.Repo
		// nolint: gocritic
		if milestone.GitHub.String() != "" {
			repo = client.Repo{
				Name:  milestone.GitHub.Name,
				Owner: milestone.GitHub.Owner,
			}
		} else if milestone.GitLab.String() != "" {
			repo = client.Repo{
				Name:  milestone.GitLab.Name,
				Owner: milestone.GitLab.Owner,
			}
		} else if milestone.Gitea.String() != "" {
			repo = client.Repo{
				Name:  milestone.Gitea.Name,
				Owner: milestone.Gitea.Owner,
			}
		}

		log.WithField("milestone", name).
			WithField("repo", repo.String()).
			Info("closing milestone")

		err = vcsClient.CloseMilestone(ctx, repo, name)

		if err != nil {
			return err
		}
	}

	return nil
}
