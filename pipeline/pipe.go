package pipeline

import "github.com/goreleaser/releaser/context"

// Pipe interface
type Pipe interface {
	// Name of the pipe
	Name() string

	// Run the pipe
	Run(ctx *context.Context) error
}
