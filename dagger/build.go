package main

import (
	"fmt"
	"runtime"
)

// Build Goreleaser
func (g *Goreleaser) Build(
	// +default="linux"
	os string,
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
	return dag.Container().
		From(fmt.Sprintf("golang:%s-alpine", g.GoVersion)). // "cgr.dev/chainguard/wolfi-base"
		WithExec([]string{"apk", "add", "go"}).
		WithExec([]string{"adduser", "-D", "nonroot"}).
		WithMountedCache("/go", dag.CacheVolume("goreleaser-goroot")).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithExec([]string{"chown", "-R", "nonroot", "/go"}).
		WithMountedCache("/gocache", dag.CacheVolume("goreleaser-gobuild")).
		WithEnvVariable("GOCACHE", "/gocache").
		WithExec([]string{"chown", "-R", "nonroot", "/gocache"}).
		WithMountedDirectory("/src", g.Source).
		WithExec([]string{"chown", "-R", "nonroot", "/src"}).
		WithWorkdir("/src")
}
