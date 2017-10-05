// Package scoop provides a Pipe that generates a scoop.sh App Manifest and pushes it to a bucket
package scoop

import (
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/pipeline"
)

// Pipe for build
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Generating Scoop Manifest"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	client, err := client.NewGitHub(ctx)
	if err != nil {
		return err
	}
	return doRun(ctx, client)
}

func doRun(ctx *context.Context, client client.Client) error {
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if ctx.Config.Scoop.Bucket.Name == "" {
		return pipeline.Skip("scoop section is not configured")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	if ctx.Config.Archive.Format == "binary" {
		return pipeline.Skip("archive format is binary")
	}

	return nil
}
