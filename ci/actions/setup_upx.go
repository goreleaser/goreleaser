package actions

import "dagger.io/dagger"

func SetupUpx(c *dagger.Container) *dagger.Container {
	return c.WithExec([]string{"apk", "add", "upx"})
}