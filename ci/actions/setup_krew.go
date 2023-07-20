package actions

import "dagger.io/dagger"

func SetupKrew(c *dagger.Container) *dagger.Container {
	goInstall := []string{"go", "install", "sigs.k8s.io/krew/cmd/validate-krew-manifest@latest"}

	return c.WithExec(goInstall)
}
