// Package After provides the pipe implementation that runs after all other pipes.
package after

import (
	"fmt"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/shell"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

func (Pipe) String() string { return "running after hooks" }

func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.After.Hooks) == 0 || skips.Any(ctx, skips.After)
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	tmpl := tmpl.New(ctx)
	/* #nosec */
	for _, step := range ctx.Config.After.Hooks {
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
