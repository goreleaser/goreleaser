package docker

import (
	"bytes"
	"context"
	"fmt"
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
func runCommand(ctx context.Context, dir, binary string, args ...string) error {
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

	log.WithFields(fields).WithField("args", args[1:]).Debug("running")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, b.String())
	}
	return nil
}
