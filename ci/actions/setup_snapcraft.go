package actions

import "dagger.io/dagger"

func SetupSnapcraft(c *dagger.Container) *dagger.Container {
	return c.WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "-yq", "--no-install-suggests", "--no-install-recommends", "install", "snapcraft"}).
		WithExec([]string{"sh", "-c", "mkdir -p $HOME/.cache/snapcraft/download"}).
		WithExec([]string{"sh", "-c", "mkdir -p $HOME/.cache/snapcraft/stage-packages"})
}
