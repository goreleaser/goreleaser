package actions

import (
	"fmt"

	"dagger.io/dagger"
)

func SetupGo(c *dagger.Container) *dagger.Container {
	version := "1.20.4"
	arch := "$(dpkg --print-architecture)"
	curl := fmt.Sprintf("curl -o go_linux.tar.gz -L https://go.dev/dl/go%s.linux-%s.tar.gz", version, arch)
	return c.WithExec([]string{"sh", "-c", curl}).
		WithExec([]string{"tar", "-C", "/usr/local", "-xvf", "go_linux.tar.gz"}).
		WithEnvVariable("PATH", "/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
}
