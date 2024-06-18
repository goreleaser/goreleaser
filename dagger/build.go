package main

import (
	"fmt"
	"runtime"
)

// Build Goreleaser
func (g *Goreleaser) Build(
	// Target OS to build
	// +default="linux"
	os string,
	// Target architecture to build
	// +optional
	arch string,
) *File {
	if arch == "" {
		arch = runtime.GOARCH
	}
	return g.BuildEnv().
		WithEnvVariable("GOOS", os).
		WithEnvVariable("GOARCH", arch).
		WithExec([]string{"go", "build", "-o", "/src/dist/goreleaser"}).
		File("/src/dist/goreleaser")
}

// Container to build Goreleaser
func (g *Goreleaser) BuildEnv() *Container {
	// Base image with Go
	env := dag.Container().
		From(fmt.Sprintf("golang:%s-alpine", g.GoVersion)). // "cgr.dev/chainguard/wolfi-base"
		WithExec([]string{"adduser", "-D", "nonroot"})

	// Mount the Go cache
	env = env.
		WithMountedCache("/go", dag.CacheVolume("goreleaser-goroot")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithExec([]string{"chown", "-R", "nonroot", "/go"})

	// Mount the Go build cache
	env = env.
		WithMountedCache("/gocache", dag.CacheVolume("goreleaser-gobuild")).
		WithEnvVariable("GOCACHE", "/gocache").
		WithExec([]string{"chown", "-R", "nonroot", "/gocache"})

	// Mount the source code
	env = env.
		WithMountedDirectory("/src", g.Source).
		WithExec([]string{"chown", "-R", "nonroot", "/src"}).
		WithWorkdir("/src")

	return env
}
