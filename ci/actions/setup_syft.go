package actions

import "dagger.io/dagger"

func SetupSyft(c *dagger.Container) *dagger.Container {
	goInstall := []string{"go", "install", "github.com/anchore/syft/cmd/syft@latest"}

	return c.WithExec(goInstall)
}