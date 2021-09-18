// Package milestone implements Pipe and manages VCS milestones.
package milestone

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const defaultNameTemplate = "{{ .Tag }}"

// Pipe for milestone.
type Pipe struct{}

func (Pipe) String() string                 { return "milestones" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Milestones) == 0 }

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Milestones {
		milestone := &ctx.Config.Milestones[i]

		if milestone.NameTemplate == "" {
			milestone.NameTemplate = defaultNameTemplate
		}

		if milestone.Repo.Name == "" {
			repo, err := git.ExtractRepoFromConfig()

			if err != nil && !ctx.Snapshot {
				return err
			}

			milestone.Repo = repo
		}
	}

	return nil
}

// Publish the release.
func (Pipe) Publish(ctx *context.Context) error {
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
			return pipe.Skip("closing not enabled")
		}

		name, err := tmpl.New(ctx).Apply(milestone.NameTemplate)
		if err != nil {
			return err
		}

		repo := client.Repo{
			Name:  milestone.Repo.Name,
			Owner: milestone.Repo.Owner,
		}

		log.WithField("milestone", name).
			WithField("repo", repo.String()).
			Info("closing milestone")

		err = vcsClient.CloseMilestone(ctx, repo, name)

		if err != nil {
			if milestone.FailOnError {
				return err
			}

			log.WithField("milestone", name).
				WithField("repo", repo.String()).
				Warnf("error closing milestone: %s", err)
		}
	}

	return nil
}
