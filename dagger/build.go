package main

import (
	"runtime"
)

const (
	// cgr.dev/chainguard/wolfi-base:latest 6/26/2024
	wolfiBase = "cgr.dev/chainguard/wolfi-base@sha256:7a5b796ae54f72b78b7fc33c8fffee9a363af2c6796dac7c4ef65de8d67d348d"
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

// Base container to build and test Goreleaser
func (g *Goreleaser) Base() *Container {
	// Base image with Go
	return dag.Container().
		From(wolfiBase).
		WithExec([]string{"apk", "add", "go"}).
		// Mount the Go cache
		WithMountedCache(
			"/go",
			dag.CacheVolume("goreleaser-goroot"),
			ContainerWithMountedCacheOpts{
				Owner: "nonroot",
			}).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		// Mount the Go build cache
		WithMountedCache(
			"/gocache",
			dag.CacheVolume("goreleaser-gobuild"),
			ContainerWithMountedCacheOpts{
				Owner: "nonroot",
			}).
		WithEnvVariable("GOCACHE", "/gocache")
}

// Container to build Goreleaser
func (g *Goreleaser) BuildEnv() *Container {
	// Base image with Go
	return g.Base().
		// Mount the source code last to optimize cache
		With(WithSource(g))
}

// Helper function to mount the project source into a container
func WithSource(g *Goreleaser) WithContainerFunc {
	return func(c *Container) *Container {
		return c.
			WithMountedDirectory("/src", g.Source, ContainerWithMountedDirectoryOpts{
				Owner: "nonroot",
			}).
			WithWorkdir("/src")
	}
}
