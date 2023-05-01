// Package before provides the pipe implementation that runs before all other pipes.
package before

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

func (Pipe) String() string { return "running before hooks" }
func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Before.Hooks) == 0 || ctx.SkipBefore
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	tmpl := tmpl.New(ctx)
	/* #nosec */
	for _, step := range ctx.Config.Before.Hooks {
		s, err := tmpl.Apply(step)
		if err != nil {
			return err
		}
		args, err := shellwords.Parse(s)
		if err != nil {
			return err
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = ctx.Env.Strings()

		var b bytes.Buffer
		w := gio.Safe(&b)
		cmd.Stderr = io.MultiWriter(logext.NewWriter(), w)
		cmd.Stdout = io.MultiWriter(logext.NewWriter(), w)

		log.WithField("hook", step).Info("running")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook failed: %s: %w; output: %s", step, err, b.String())
		}
	}
	return nil
}
