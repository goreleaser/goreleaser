// A module for Goreleaser Dagger functions

package main

import (
	"context"
	"fmt"
	"runtime"
)

type Goreleaser struct {
	// +private
	Source *Directory
	// +private
	GoVersion string
}

func New(
	// The Goreleaser source code to use
	// +optional
	Source *Directory,
	// The Go version to use // TODO: look up default based on "stable"
	// +default="1.22.3"
	GoVersion string,
) *Goreleaser {
	// TODO: remove
	if Source == nil {
		Source = dag.Git(
			"https://github.com/goreleaser/goreleaser.git",
			GitOpts{KeepGitDir: true},
		).
			Branch("main").
			Tree()
	}
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
		WithExec([]string{
			"golangci-lint",
			"run",
			"--config",
			"./.golangci.yaml",
			"./...",
		}).
		Stdout(ctx)
}

// Test Goreleaser
func (g *Goreleaser) Test(ctx context.Context) (string, error) {
	return g.TestEnv().
		WithExec([]string{
			"go",
			"test",
			"-failfast",
			// "-race", // TODO: change base
			"-coverpkg=./...",
			"-covermode=atomic",
			"-coverprofile=coverage.txt",
			"./...",
			"-run",
			".",
		}).
		Stdout(ctx)
}

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

// Container to build Goreleaser
func (g *Goreleaser) BuildEnv() *Container {
	return dag.Container().
		From(fmt.Sprintf("golang:%s-alpine", g.GoVersion)). // "cgr.dev/chainguard/wolfi-base"
		WithExec([]string{"apk", "add", "go"}).
		WithExec([]string{"adduser", "-D", "nonroot"}).
		// WithMountedCache("/go/pkg/mod", dag.CacheVolume("goreleaser-gomod")). // TODO: fix caching
		// WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		// WithExec([]string{"chown", "-R", "nonroot", "/go"}).
		// WithMountedCache("/gocache", dag.CacheVolume("goreleaser-gobuild")).
		// WithEnvVariable("GOCACHE", "/gocache").
		// WithExec([]string{"chown", "-R", "nonroot", "/gocache"}).
		WithMountedDirectory("/src", g.Source).
		WithExec([]string{"chown", "-R", "nonroot", "/src"}).
		WithWorkdir("/src")
}

// Container to test Goreleaser
func (g *Goreleaser) TestEnv() *Container {
	// install krew
	// install snapcraft
	// install tparse
	return g.BuildEnv().
		// WithEnvVariable("CGO_ENABLED", "1"). // TODO: change base
		WithServiceBinding("localhost", dag.Docker().Engine()). // TODO: fix localhost
		WithEnvVariable("DOCKER_HOST", "tcp://localhost:2375").
		WithExec(
			[]string{"apk", "add",
				"bash",
				"curl",
				"git",
				"gpg",
				"gpg-agent",
				"nix",
				"upx",
				"cosign",
				"docker",
				"syft",
			}).
		// WithExec([]string{"sh", "-c", "sh <(curl -L https://nixos.org/nix/install) --no-daemon"})
		WithExec([]string{"chown", "-R", "nonroot", "/nix"}).
		WithUser("nonroot").
		WithExec([]string{"go", "install", "github.com/google/ko@latest"})
}
