package pipeline

import (
	"fmt"

	"github.com/goreleaser/goreleaser/context"
)

// Piper defines a pipe, which can be part of a pipeline (a serie of pipes).
type Piper interface {
	fmt.Stringer

	// Run the pipe
	Run(ctx *context.Context) error
}

// IsSkip returns true if the error is an ErrSkip
func IsSkip(err error) bool {
	_, ok := err.(ErrSkip)
	return ok
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
