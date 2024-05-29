// A module for Goreleaser Dagger functions

package main

import (
	"context"
	"fmt"
)

type Goreleaser struct {
	Source *Directory
}

func New(
	// The Goreleaser source code to use
	Source *Directory,
) *Goreleaser {
	return &Goreleaser{Source: Source}
}

func (g *Goreleaser) Lint(
	ctx context.Context,
	// +optional
	// +default="v1.58.1"
	golangciLintVersion string,
) (string, error) {
	lintImage := fmt.Sprintf("golangci/golangci-lint:%s", golangciLintVersion)
	return dag.Container().From(lintImage).
		WithMountedDirectory("/src", g.Source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "--config", "./.golangci.yaml", "./..."}).
		Stdout(ctx)
}

// Returns a container that echoes whatever string argument is provided
func (m *Goreleaser) ContainerEcho(stringArg string) *Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *Goreleaser) GrepDir(ctx context.Context, directoryArg *Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}
