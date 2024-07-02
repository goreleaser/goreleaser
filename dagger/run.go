package main

import (
	"context"
	"runtime"
)

// Run Goreleaser
func (g *Goreleaser) Run(
	ctx context.Context,
	// Context directory to run in
	context *Directory,
	// Arguments to pass to Goreleaser
	args []string,
) (string, error) {
	binary := g.Build("linux", runtime.GOARCH)

	return dag.Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithMountedFile("/bin/goreleaser", binary).
		WithMountedDirectory("/src", context).
		WithWorkdir("/src").
		WithExec(append([]string{"/bin/goreleaser"}, args...)).
		Stdout(ctx)
}
