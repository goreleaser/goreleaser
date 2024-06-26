package main

import (
	"context"
	"fmt"
	"runtime"
)

// Test Goreleaser
func (g *Goreleaser) Test(ctx context.Context) *Container {
	return g.TestEnv().
		WithExec([]string{
			"go",
			"test",
			"-failfast",
			"-race",
			"-coverpkg=./...",
			"-covermode=atomic",
			"-coverprofile=coverage.txt",
			"./...",
			"-run",
			".",
		})
}

// Container to test Goreleaser
func (g *Goreleaser) TestEnv() *Container {
	// Dependencies needed for testing
	testDeps := []string{
		"bash",
		"curl",
		"git",
		"gpg",
		"gpg-agent",
		"upx",
		"cosign",
		"docker",
		"syft",
	}
	return g.BuildEnv().
		WithEnvVariable("CGO_ENABLED", "1").
		WithServiceBinding("localhost", dag.Docker().Engine()).
		WithEnvVariable("DOCKER_HOST", "tcp://localhost:2375").
		WithExec(append([]string{"apk", "add"}, testDeps...)).
		With(installNix).
		With(installBuildx).
		WithUser("nonroot").
		WithExec([]string{"go", "install", "github.com/google/ko@latest"})
}

func installNix(target *Container) *Container {
	nix := dag.Container().From("nixos/nix:2.18.3")
	nixBin := "/root/.nix-profile/bin"

	binaries := []string{
		"nix",
		"nix-build",
		"nix-channel",
		"nix-collect-garbage",
		"nix-copy-closure",
		"nix-daemon",
		"nix-env",
		"nix-hash",
		"nix-instantiate",
		"nix-prefetch-url",
		"nix-shell",
		"nix-store",
	}

	for _, binary := range binaries {
		target = target.WithFile("/bin/"+binary, nix.File(nixBin+"/"+binary))
	}

	target = target.WithDirectory("/nix/store", nix.Directory("/nix/store"))

	return target
}

func installBuildx(target *Container) *Container {
	arch := runtime.GOARCH
	url := fmt.Sprintf("https://github.com/docker/buildx/releases/download/v0.15.1/buildx-v0.15.1.linux-%s", arch)

	bin := dag.HTTP(url)

	return target.WithFile(
		"/usr/lib/docker/cli-plugins/docker-buildx",
		bin,
		ContainerWithFileOpts{
			Permissions: 0777,
		})
}
