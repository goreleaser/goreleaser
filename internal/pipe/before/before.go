// Package before provides the pipe implementation that runs before all other pipes.
package before

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/apex/log"
	"github.com/caarlos0/go-shellwords"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "running before hooks"
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
		log := log.WithField("hook", step)
		log.Info("running")
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = ctx.Env.Strings()
		var b bytes.Buffer
		cmd.Stderr = io.MultiWriter(logext.NewErrWriter(log), &b)
		cmd.Stdout = io.MultiWriter(logext.NewWriter(log), &b)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook failed: %s: %w; output: %s", step, err, b.String())
		}
	}
	return nil
}
