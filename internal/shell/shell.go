// Package shell handles shell commands.
package shell

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/redact"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Run a shell command with given arguments and envs
func Run(ctx *context.Context, dir string, command, env []string, output bool) error {
	if len(command) == 0 {
		log.Warn("skipping empty command")
		return nil
	}

	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env

	var b bytes.Buffer
	w := gio.Safe(&b)

	cmd.Stderr = redact.Writer(io.MultiWriter(logext.NewConditionalWriter(output), w), env)
	cmd.Stdout = redact.Writer(io.MultiWriter(logext.NewConditionalWriter(output), w), env)

	if dir != "" {
		cmd.Dir = dir
	}

	log.WithField("cmd", command).
		WithField("dir", dir).
		Debug("running")

	start := time.Now()
	defer logext.Duration(start, time.Second*5)

	if err := cmd.Run(); err != nil {
		return gerrors.Wrap(
			err,
			gerrors.WithMessage("command failed"),
			gerrors.WithDetails("cmd", command[0]),
			gerrors.WithOutput(strings.TrimSpace(b.String())),
		)
	}

	return nil
}
