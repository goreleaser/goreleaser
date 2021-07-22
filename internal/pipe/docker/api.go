package docker

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/logext"
)

var (
	manifesters = map[string]manifester{}
	imagers     = map[string]imager{}
	lock        sync.Mutex
)

func registerManifester(use string, impl manifester) {
	lock.Lock()
	defer lock.Unlock()
	manifesters[use] = impl
}

func registerImager(use string, impl imager) {
	lock.Lock()
	defer lock.Unlock()
	imagers[use] = impl
}

// imager is something that can build and push docker images.
type imager interface {
	Build(ctx context.Context, root string, images, flags []string) error
	Push(ctx context.Context, image string, flags []string) error
}

// manifester is something that can create and push docker manifests.
type manifester interface {
	Create(ctx context.Context, manifest string, images, flags []string) error
	Push(ctx context.Context, manifest string, flags []string) error
}

// nolint: unparam
func runCommand(ctx context.Context, dir, binary string, args ...string) ([]byte, error) {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	log := log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir)
	var b bytes.Buffer
	cmd.Stderr = io.MultiWriter(logext.NewErrWriter(log), &b)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(log), &b)

	log.Debug("running")
	return b.Bytes(), cmd.Run()
}
