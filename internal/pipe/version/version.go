package version

import (
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe is a global hook pipe.
type Pipe struct{}

// String is the name of this pipe.
func (Pipe) String() string {
	return "overriding version"
}

// Run executes the hooks.
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Version != "" {
		version, err := tmpl.New(ctx).Apply(ctx.Config.Version)
		if err != nil {
			return err
		}
		ctx.Version = version
	}
	return nil
}
