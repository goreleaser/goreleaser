// Package pipeline provides a generic pipe interface.
package pipeline

import "github.com/goreleaser/goreleaser/context"

// Pipe interface
type Pipe interface {
	// Name of the pipe
	Description() string

	// Run the pipe
	Run(ctx *context.Context) error
}

// ErrSkip occurs when a pipe is skipped for some reason
type ErrSkip struct {
	reason string
}

// Error implements the error interface. returns the reason the pipe was skipped
func (e ErrSkip) Error() string {
	return e.reason
}

// Skip skips this pipe with the given reason
func Skip(reason string) ErrSkip {
	return ErrSkip{reason}
}
