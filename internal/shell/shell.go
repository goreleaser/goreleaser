package shell

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/apex/log"

	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Run a shell command with given arguments and envs
func Run(ctx *context.Context, dir string, command, env []string) error {
	fields := log.Fields{
		"cmd": command,
		"env": env,
	}

	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env

	var b bytes.Buffer
	w := gio.Safe(&b)

	cmd.Stderr = io.MultiWriter(logext.NewWriter(fields, logext.Error), w)
	cmd.Stdout = io.MultiWriter(logext.NewWriter(fields, logext.Info), w)

	if dir != "" {
		cmd.Dir = dir
	}

	log.WithFields(fields).Debug("running")
	if err := cmd.Run(); err != nil {
		log.WithFields(fields).WithError(err).Debug("failed")
		return fmt.Errorf("%q: %w", b.String(), err)
	}

	return nil
}
