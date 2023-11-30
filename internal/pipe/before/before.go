// Package before provides the pipe implementation that runs before all other pipes.
package before

import (
	"fmt"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

func (Pipe) String() string { return "running before hooks" }

func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Before.Hooks) == 0 || skips.Any(ctx, skips.Before)
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

		log.WithField("hook", s).Info("running")
		if err := shell.Run(ctx, "", args, ctx.Env.Strings(), false); err != nil {
			return fmt.Errorf("hook failed: %w", err)
		}
	}
	return nil
}
