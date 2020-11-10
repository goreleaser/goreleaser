// Package before provides the pipe implementation that runs before all other pipes.
package before

import (
	"fmt"
	"os/exec"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/mattn/go-shellwords"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "running before hooks"
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	var tmpl = tmpl.New(ctx)
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
		log.Infof("running %s", color.CyanString(step))
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = ctx.Env.Strings()
		out, err := cmd.CombinedOutput()
		log.WithField("cmd", step).Debug(string(out))
		if err != nil {
			return fmt.Errorf("hook failed: %s: %w; output: %s", step, err, string(out))
		}
	}
	return nil
}
