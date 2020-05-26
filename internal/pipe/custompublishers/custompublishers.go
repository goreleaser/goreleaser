// Package custompublishers provides a Pipe that executes a custom publisher
package custompublishers

import (
	"github.com/goreleaser/goreleaser/internal/exec"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for custom publisher.
type Pipe struct{}

// String returns the description of the pipe.
func (Pipe) String() string {
	return "custom publisher"
}

// Publish artifacts.
func (Pipe) Publish(ctx *context.Context) error {
	if len(ctx.Config.Publishers) == 0 {
		return pipe.Skip("publishers section is not configured")
	}

	return exec.Execute(ctx, ctx.Config.Publishers)
}
