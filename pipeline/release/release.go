// Package release implements Pipe and manages github releases and its
// artifacts.
package release

import (
	"os"

	"github.com/goreleaser/goreleaser/internal/artifact"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

func (Pipe) String() string {
	return "releasing to GitHub"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Release.NameTemplate == "" {
		ctx.Config.Release.NameTemplate = "{{.Tag}}"
	}
	if ctx.Config.Release.GitHub.Name != "" {
		return nil
	}
	repo, err := remoteRepo()
	if err != nil {
		return err
	}
	ctx.Config.Release.GitHub = repo
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	c, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, c)
}

func doRun(ctx *context.Context, c client.Client) error {
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	log.WithField("tag", ctx.Git.CurrentTag).
		WithField("repo", ctx.Config.Release.GitHub.String()).
		Info("creating or updating release")
	body, err := describeBody(ctx)
	if err != nil {
		return err
	}
	releaseID, err := c.CreateRelease(ctx, body.String())
	if err != nil {
		return err
	}
	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, artifact := range ctx.Artifacts.Filter(
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
			artifact.ByType(artifact.Checksum),
			artifact.ByType(artifact.Signature),
		),
	).List() {
		sem <- true
		artifact := artifact
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return upload(ctx, c, releaseID, artifact)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, c client.Client, releaseID int, artifact artifact.Artifact) error {
	file, err := os.Open(artifact.Path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.WithField("file", file.Name()).WithField("name", artifact.Name).Info("uploading to release")
	return c.Upload(ctx, releaseID, artifact.Name, file)
}
