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

	stderr := redact.Writer(io.MultiWriter(logext.NewConditionalWriter(output), w), env)
	stdout := redact.Writer(io.MultiWriter(logext.NewConditionalWriter(output), w), env)
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	if dir != "" {
		cmd.Dir = dir
	}

	log.WithField("cmd", redactArgs(command, cmd.Env)).
		WithField("dir", dir).
		Debug("running")

	start := time.Now()
	defer logext.Duration(start, time.Second*5)

	runErr := cmd.Run()
	stderrErr := stderr.Close()
	stdoutErr := stdout.Close()
	if runErr != nil {
		return gerrors.Wrap(
			runErr,
			gerrors.WithMessage("command failed"),
			gerrors.WithDetails("cmd", command[0]),
			gerrors.WithOutput(strings.TrimSpace(b.String())),
		)
	}
	if stderrErr != nil {
		return stderrErr
	}
	if stdoutErr != nil {
		return stdoutErr
	}

	return nil
}

func redactArgs(args, env []string) []string {
	redacted := make([]string, len(args))
	for i, arg := range args {
		redacted[i] = redact.String(arg, env)
	}
	return redacted
}
