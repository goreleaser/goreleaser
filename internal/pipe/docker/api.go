package docker

import (
	"context"
	"os/exec"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// imager is something that can build and push docker images.
type imager interface {
	Build(ctx context.Context, root string, images, flags []string) error
	Push(ctx context.Context, image string) error
}

// manifester is something that can create and push docker manifests.
type manifester interface {
	Create(ctx context.Context, manifest string, images, flags []string) error
	Push(ctx context.Context, manifest string, flags []string) error
}

func newManifester(manifest config.DockerManifest) manifester {
	return dockerManifester{}
}

func newImager(docker config.Docker) imager {
	return dockerImager{
		buildx: docker.Use == useBuildx,
	}
}

// nolint: unparam
func runCommand(ctx context.Context, dir, binary string, args ...string) error {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	log := log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir)
	log.Debug("running")
	out, err := cmd.CombinedOutput()
	log.Debug(string(out))
	return err
}
