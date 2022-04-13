package docker

import (
	"fmt"
	"github.com/goreleaser/goreleaser/pkg/config"
	"os"

	"github.com/goreleaser/goreleaser/pkg/context"
)

func init() {
	registerImager(useKaniko, kanikoImager{})
}

const (
	kanikoExecutorImage = "gcr.io/kaniko-project/executor:latest"
	kanikoWorkspace     = "/workspace"
)

type kanikoImager struct{}

// Push is a no-op and it returns nil because kaniko performs build and push in a single step.
// https://github.com/GoogleContainerTools/kaniko#how-does-kaniko-work
func (i kanikoImager) Push(_ *context.Context, _ string, _ []string) error {
	return nil
}

func (i kanikoImager) Build(ctx *context.Context, docker config.Docker, root string, images, flags []string) error {
	if err := runCommand(ctx, root, "docker", i.buildCommand(docker, images, flags)...); err != nil {
		return fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	return nil
}

func (i kanikoImager) buildCommand(docker config.Docker, images, flags []string) []string {
	base := []string{"run"}

	base = i.appendWorkspaceVolumeMount(base)
	base = i.appendKanikoExectorImage(base)
	base = i.appendContext(base)
	base = i.appendDockerfile(base, docker.Dockerfile)
	base = i.appendDestinations(base, images)
	base = i.appendNoPush(base, docker.SkipPush == "true")
	base = i.appendFlags(base, flags)

	return base
}

func (i kanikoImager) appendWorkspaceVolumeMount(base []string) []string {
	cwd, _ := os.Getwd()
	return append(base, "-v", fmt.Sprintf("%s:%s", cwd, kanikoWorkspace))
}

func (i kanikoImager) appendKanikoExectorImage(base []string) []string {
	return append(base, kanikoExecutorImage)
}

func (i kanikoImager) appendContext(base []string) []string {
	return append(base, "--context", fmt.Sprintf("dir://%s", kanikoWorkspace))
}

func (i kanikoImager) appendDockerfile(base []string, dockerfile string) []string {
	return append(base, "--dockerfile", fmt.Sprintf("%s/%s", kanikoWorkspace, dockerfile))
}

func (i kanikoImager) appendDestinations(base, images []string) []string {
	for _, image := range images {
		base = append(base, "--destination", image)
	}
	return base
}

func (i kanikoImager) appendNoPush(base []string, skipPush bool) []string {
	if skipPush {
		base = append(base, "--no-push")
	}
	return base
}

func (i kanikoImager) appendFlags(base []string, flags []string) []string {
	return append(base, flags...)
}
