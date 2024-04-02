package builder

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// nolint: unparam
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

func runCommandWithOutput(ctx *context.Context, dir, binary string, args ...string) ([]byte, []byte, error) {
	/* #nosec */
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(ctx.Env.Strings(), cmd.Environ()...)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	var b bytes.Buffer
	w := gio.Safe(&b)
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
		return nil, errBuf.Bytes(), fmt.Errorf("%w: %s", err, b.String())
	}

	return out, errBuf.Bytes(), nil
}
