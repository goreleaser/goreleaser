package main

import (
	"context"
	"fmt"
	"runtime"
)

const (
	// nixos/nix:2.18.3
	nixBase       = "nixos/nix@sha256:3f8fec6acf10ae6f1c8843ccc607490d4635d620a75afc0b523eedba7617c16e"
	buildxVersion = "v0.15.1"
)

// Test Goreleaser
func (g *Goreleaser) Test(ctx context.Context) *TestResult {
	test := g.TestEnv().
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

	return &TestResult{
		Container: test,
	}
}

// Custom type for test results
type TestResult struct {
	// Container with the test executed
	Container *Container
}

// Coverage report from the test. Save with '-o ./coverage.txt'
func (t *TestResult) CoverageReport() *File {
	return t.Container.File("coverage.txt")
}

// Stdout from the test command
func (t *TestResult) Output(ctx context.Context) (string, error) {
	return t.Container.Stdout(ctx)
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
	return g.Base().
		WithEnvVariable("CGO_ENABLED", "1").
		WithExec(append([]string{"apk", "add"}, testDeps...)).
		With(installNix).
		With(installBuildx).
		WithUser("nonroot").
		WithExec([]string{"go", "install", "github.com/google/ko@latest"}).
		// This is bound at localhost for the hardcoded docker and ko registry tests
		WithServiceBinding("localhost", dag.Docker().Engine()).
		WithEnvVariable("DOCKER_HOST", "tcp://localhost:2375").
		// Mount the source code last to optimize cache
		With(WithSource(g))
}

// Install Nix binaries from nixos image
func installNix(target *Container) *Container {
	nix := dag.Container().From(nixBase)
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

// Install buildx plugin for Docker from buildx github release
func installBuildx(target *Container) *Container {
	arch := runtime.GOARCH
	url := fmt.Sprintf("https://github.com/docker/buildx/releases/download/%s/buildx-%s.linux-%s", buildxVersion, buildxVersion, arch)

	bin := dag.HTTP(url)

	return target.WithFile(
		"/usr/lib/docker/cli-plugins/docker-buildx",
		bin,
		ContainerWithFileOpts{
			Permissions: 0777,
		})
}
