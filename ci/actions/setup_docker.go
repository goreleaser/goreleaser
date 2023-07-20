package actions

import (
	"fmt"

	"dagger.io/dagger"
)

// echo "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

func SetupDocker(c *dagger.Container) *dagger.Container {
	version := "20.10.24"
	version_apt := fmt.Sprintf("5:%s~3-0~ubuntu-jammy", version)
	install := fmt.Sprintf("apt-get install -y docker-ce=%s docker-ce-cli=%s containerd.io docker-buildx-plugin docker-compose-plugin", version_apt, version_apt)
	return c.
		WithExec([]string{"apt-get", "install", "ca-certificates", "curl", "gnupg"}).
		WithExec([]string{"install", "-m", "0755", "-d", "/etc/apt/keyrings"}).
		WithExec([]string{"sh", "-c", "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"}).
		WithExec([]string{"chmod", "a+r", "/etc/apt/keyrings/docker.gpg"}).
		WithExec([]string{"sh", "-c", `echo "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null`}).
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"sh", "-c", install})
}
