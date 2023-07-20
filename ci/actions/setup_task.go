package actions

import "dagger.io/dagger"

func SetupTask(c *dagger.Container) *dagger.Container {
	goInstall := []string{"go", "install", "github.com/go-task/task/v3/cmd/task@latest"}

	return c.WithExec(goInstall)
}
