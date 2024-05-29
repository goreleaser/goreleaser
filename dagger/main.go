// A module for Goreleaser Dagger functions

package main

import (
	"context"
	"fmt"
)

type Goreleaser struct {
	Source    *Directory
	GoVersion string
}

func New(
	// The Goreleaser source code to use
	Source *Directory,
	// The Go version to use
	// +default="1.22.3"
	GoVersion string,
) *Goreleaser {
	return &Goreleaser{Source: Source, GoVersion: GoVersion}
}

// Lint Goreleaser
func (g *Goreleaser) Lint(
	ctx context.Context,
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

// Test Goreleaser
func (g *Goreleaser) Test(ctx context.Context) (string, error) {
	return g.TestEnv().
		WithExec([]string{"go", "test", "./..."}).
		Stdout(ctx)
}

// Build Goreleaser
func (g *Goreleaser) Build() *File {
	return g.BuildEnv().
		WithExec([]string{"go", "build", "-o", "/src/dist/goreleaser"}).
		File("/src/dist/goreleaser")
}

// Build environment to build Goreleaser
func (g *Goreleaser) BuildEnv() *Container {
	return dag.Container().
		From(fmt.Sprintf("golang:%s-bullseye", g.GoVersion)).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("goreleaser-gomod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("goreleaser-gobuild")).
		WithMountedDirectory("/src", g.Source).
		WithWorkdir("/src")
}

// Test environment to test Goreleaser
func (g *Goreleaser) TestEnv() *Container {
	return g.BuildEnv()
}
