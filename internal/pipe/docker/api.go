package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// imager is something that can build and push docker images.
type imager interface {
	Build(ctx context.Context, root string, images, flags []string) error
	Push(ctx context.Context, image string) error
}

type manifester interface {
	Remove(ctx context.Context, manifest string) error
}

func newManifester(manifest config.DockerManifest) manifester {
	return dockerManifester{
		binary: useDocker,
	}
}

func newImager(docker config.Docker) imager {
	binary := useDocker
	if docker.Use == usePodman {
		binary = usePodman
	}
	return dockerImager{
		buildx: docker.Use == useBuildx,
		binary: binary,
	}
}

type dockerManifester struct{}

func (m dockerManifester) Remove(ctx context.Context, manifest string) error {
	log.WithField("manifest", manifest).Info("removing local docker manifest")
	/* #nosec */
	cmd := exec.CommandContext(ctx, "docker", "manifest", "rm", manifest)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.HasPrefix(string(out), "No such manifest: ") {
			// ignore "no such manifest" error, is the state we want in the end...
			return nil
		}
		return fmt.Errorf("failed to remove local docker manifest: %s: \n%s: %w", manifest, string(out), err)
	}
	log.Debugf("docker manifest rm output: \n%s", string(out))
	return nil
}

type dockerImager struct {
	buildx bool
	binary string
}

func (i dockerImager) Push(ctx context.Context, image string) error {
	log.WithField("image", image).Info("pushing docker image")
	/* #nosec */
	cmd := exec.CommandContext(ctx, i.binary, "push", image)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push docker image: \n%s: %w", string(out), err)
	}
	log.Debugf("%s push output: \n%s", i.binary, string(out))
	return nil
}

func (i dockerImager) Build(ctx context.Context, root string, images, flags []string) error {
	cmd := exec.CommandContext(ctx, i.binary, i.buildCommand(images, flags)...)
	cmd.Dir = root
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build docker image: %s: \n%s: %w", images[0], string(out), err)
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func (i dockerImager) buildCommand(images, flags []string) []string {
	base := []string{"build", "."}
	if i.buildx {
		base = []string{"buildx", "build", ".", "--load"}
	}
	for _, image := range images {
		base = append(base, "-t", image)
	}
	base = append(base, flags...)
	return base
}
