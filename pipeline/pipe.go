package pipeline

import "github.com/goreleaser/goreleaser/context"

// Pipe interface
type Pipe interface {
	// Name of the pipe
	Description() string

	// Run the pipe
	Run(ctx *context.Context) error
}
