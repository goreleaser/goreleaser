package shell

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Run a shell command with given arguments and envs
func Run(ctx *context.Context, dir string, command, env []string, output bool) error {
	log := log.
		WithField("cmd", command).
		WithField("env", env)

	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env

	var b bytes.Buffer
	w := gio.Safe(&b)

	cmd.Stderr = io.MultiWriter(logext.NewConditionalWriter(output), w)
	cmd.Stdout = io.MultiWriter(logext.NewConditionalWriter(output), w)

	if dir != "" {
		cmd.Dir = dir
	}

	log.Debug("running")
	if err := cmd.Run(); err != nil {
		log.WithError(err).Debug("failed")
		return fmt.Errorf("failed to run '%s': %w", strings.Join(command, " "), err)
	}

	return nil
}
