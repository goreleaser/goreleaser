package docker

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	Build(ctx *context.Context, root string, images, flags []string) error
	Push(ctx *context.Context, image string, flags []string) (digest string, err error)
}

// manifester is something that can create and push docker manifests.
type manifester interface {
	Create(ctx *context.Context, manifest string, images, flags []string) error
	Push(ctx *context.Context, manifest string, flags []string) (digest string, err error)
}

func runCommand(ctx *context.Context, dir, binary string, args ...string) error {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)

	log.
		WithField("cmd", append([]string{binary}, args[0])).
		WithField("cwd", dir).
		WithField("args", args[1:]).Debug("running")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, b.String())
	}
	return nil
}

func runCommandWithOutput(ctx *context.Context, dir, binary string, args ...string) ([]byte, error) {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)

	var b bytes.Buffer
	w := gio.Safe(&b)
	cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)

	log.
		WithField("cmd", append([]string{binary}, args[0])).
		WithField("cwd", dir).
		WithField("args", args[1:]).
		Debug("running")
	out, err := cmd.Output()
	if out != nil {
		// regardless of command success, always print stdout for backward-compatibility with runCommand()
		_, _ = io.MultiWriter(logext.NewWriter(), w).Write(out)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, b.String())
	}

	return out, nil
}
