// Package release implements Pipe and manages github releases and its
// artifacts.
package release

import (
	"log"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/client"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for github release
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Releasing to GitHub"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	return doRun(ctx, client.NewGitHub(ctx))
}

func doRun(ctx *context.Context, client client.Client) error {
	if !ctx.Publish {
		return nil
	}
	log.Println("Creating or updating release", ctx.Git.CurrentTag, "on", ctx.Config.Release.GitHub.String())
	body, err := buildBody(ctx)
	if err != nil {
		return err
	}
	releaseID, err := client.CreateRelease(ctx, body.String())
	if err != nil {
		return err
	}
	var g errgroup.Group
	for _, artifact := range ctx.Artifacts {
		artifact := artifact
		g.Go(func() error {
			return upload(ctx, client, releaseID, artifact)
		})
	}
	return g.Wait()
}

func upload(ctx *context.Context, client client.Client, releaseID int, artifact string) error {
	var path = filepath.Join(ctx.Config.Dist, artifact)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	log.Println("Uploading", file.Name())
	return client.Upload(ctx, releaseID, artifact, file)
}
