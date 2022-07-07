// Package custompublishers provides a Pipe that executes a custom publisher
package custompublishers

import (
	"github.com/goreleaser/goreleaser/internal/exec"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for custom publisher.
type Pipe struct{}

// String returns the description of the pipe.
func (Pipe) String() string                 { return "custom publisher" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Publishers) == 0 }

// Publish artifacts.
func (Pipe) Publish(ctx *context.Context) error {
	return exec.Execute(ctx, ctx.Config.Publishers)
}
