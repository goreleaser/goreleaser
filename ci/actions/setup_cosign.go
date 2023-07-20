package actions

import "dagger.io/dagger"

func SetupCosign(c *dagger.Container) *dagger.Container {
	goInstall := []string{"go", "install", "github.com/sigstore/cosign/v2/cmd/cosign@latest"}

	return c.WithExec(goInstall)
}
