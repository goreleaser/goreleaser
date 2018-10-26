package release

import (
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for github release
type Pipe struct{}

func (Pipe) String() string {
	return "releasing to GitHub"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Release.Disable {
		return nil
	}
	if ctx.Config.Release.NameTemplate == "" {
		ctx.Config.Release.NameTemplate = "{{.Tag}}"
	}
	if ctx.Config.Release.GitHub.Name != "" {
		return nil
	}
	repo, err := remoteRepo()
	if err != nil && !ctx.Snapshot {
		return err
	}
	ctx.Config.Release.GitHub = repo
	return nil
}

// Publish github release
func (Pipe) Publish(ctx *context.Context) error {
	c, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doPublish(ctx, c)
}

func doPublish(ctx *context.Context, c client.Client) error {
	if ctx.Config.Release.Disable {
		return pipe.Skip("release pipe is disabled")
	}
	log.WithField("tag", ctx.Git.CurrentTag).
		WithField("repo", ctx.Config.Release.GitHub.String()).
		Info("release")
	body, err := describeBody(ctx)
	if err != nil {
		return err
	}
	releaseID, err := c.CreateRelease(ctx, body.String())
	if err != nil {
		return err
	}
	var g = semerrgroup.New(ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
			artifact.ByType(artifact.Checksum),
			artifact.ByType(artifact.Signature),
			artifact.ByType(artifact.LinuxPackage),
		),
	).List() {
		artifact := artifact
		g.Go(func() error {
			return upload(ctx, c, releaseID, artifact)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, c client.Client, releaseID int64, artifact artifact.Artifact) error {
	file, err := os.Open(artifact.Path)
	if err != nil {
		return err
	}
	defer file.Close() // nolint: errcheck
	log.WithField("file", file.Name()).WithField("name", artifact.Name).Info("uploading to release")
	return c.Upload(ctx, releaseID, artifact.Name, file)
}
