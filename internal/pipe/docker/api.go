package docker

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/gio"
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
	fields := log.Fields{
		"cmd": append([]string{binary}, args[0]),
		"cwd": dir,
	}

	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)

	log.WithFields(fields).Debug("running")
	err := cmd.Run()
	return b.Bytes(), err
}
