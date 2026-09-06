// Package prebuild provides a pipe that runs before the build and gomod pipes, mainly to resolve common templates.
package prebuild

import (
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe for build.
type Pipe struct{}

func (Pipe) String() string {
	return "build prerequisites"
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	tpl := tmpl.New(ctx)
	for i := range ctx.Config.Builds {
		// the node builder renders Main per target, because it accepts
		// target-aware fields such as .Target, .Os and .Arch.
		if ctx.Config.Builds[i].Builder == "node" {
			continue
		}
		m, err := tpl.Apply(ctx.Config.Builds[i].Main)
		if err != nil {
			return err
		}
		if m == "" {
			m = "."
		}
		ctx.Config.Builds[i].Main = m
	}
	return nil
}
