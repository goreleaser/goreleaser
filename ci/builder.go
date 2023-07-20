package main

import (
	"dagger.io/dagger"
	"github.com/goreleaser/goreleaser/ci/actions"
)

// TODO: break up and comment
func builder(client *dagger.Client, source *dagger.Directory) *dagger.Container {
	return client.Container().Pipeline("build").
		From("ubuntu:jammy@sha256:83f0c2a8d6f266d687d55b5cb1cb2201148eb7ac449e4202d9646b9083f1cee0").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "git", "gpg", "nix-bin", "upx-ucl"}).
		With(actions.SetupDocker).
		WithExec([]string{"adduser", "-q", "nonroot"}).
		WithMountedCache("/gomods", client.CacheVolume("gomodcache")).
		WithExec([]string{"chown", "nonroot", "/gomods"}).
		WithEnvVariable("GOMODCACHE", "/gomods").
		WithMountedCache("/gocache", client.CacheVolume("gobuildcache")).
		WithExec([]string{"chown", "nonroot", "/gocache"}).
		WithEnvVariable("GOCACHE", "/gocache").
		WithExec([]string{"mkdir", "/nix"}).
		WithExec([]string{"chown", "nonroot", "/nix"}).
		WithMountedDirectory("/src", source).
		WithExec([]string{"chown", "-R", "nonroot", "/src"}).
		WithNewFile("/usr/local/sbin/docker", dagger.ContainerWithNewFileOpts{
			Contents: `#!/bin/sh
		DOCKER_HOST=tcp://localhost:2375 /usr/bin/docker $@`,
			Permissions: 0o777,
		}).
		With(actions.SetupGo).
		WithUser("nonroot").
		WithWorkdir("/src").
		With(actions.SetupTask).
		With(actions.SetupCosign).
		With(actions.SetupSyft).
		With(actions.SetupKrew).
		WithServiceBinding("localhost", dockerd(client)).
		WithEnvVariable("DOCKER_HOST", "tcp://localhost:2375")
}

func dockerd(client *dagger.Client) *dagger.Container {
	return client.Container().Pipeline("dockerd").From("docker:20-dind"). // TODO: shared docker cache
										WithExposedPort(2375).
										WithExec([]string{"dockerd", "--log-level=error", "--host=tcp://0.0.0.0:2375", "--tls=false"}, dagger.ContainerWithExecOpts{InsecureRootCapabilities: true})
}
