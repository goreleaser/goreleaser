package docker

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
		WithField("args", args[1:]).
		Debug("running")
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

// doWithRetry performs an operation with configurable retry logic.
func doWithRetry[T any](retry config.Retry, fn func() (T, error), isRetryable func(error) bool, name string) (T, error) {
	var zero T
	var try int
	for try < retry.Max {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		if !isRetryable(err) {
			return zero, fmt.Errorf("failed to %s after %d tries: %w", name, try+1, err)
		}
		log.WithField("try", try).
			WithError(err).
			Warnf("failed to %s, will retry", name)
		time.Sleep(min(time.Duration(try+1)*retry.InitialInterval, retry.MaxInterval))
		try++
	}
	return zero, fmt.Errorf("failed to %s after %d tries", name, retry.Max)
}
