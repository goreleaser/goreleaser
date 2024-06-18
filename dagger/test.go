package main

import "context"

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

// Container to test Goreleaser
func (g *Goreleaser) TestEnv() *Container {
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
