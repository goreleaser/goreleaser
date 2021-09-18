// Package skip can skip an entire Action.
package skip

import (
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Skipper defines a method to skip an entire Piper.
type Skipper interface {
	// Skip returns true if the Piper should be skipped.
	Skip(ctx *context.Context) bool
}

// Maybe returns an action that skips immediately if the given p is a Skipper
// and its Skip method returns true.
func Maybe(skipper interface{}, next middleware.Action) middleware.Action {
	if skipper, ok := skipper.(Skipper); ok {
		return func(ctx *context.Context) error {
			if skipper.Skip(ctx) {
				return nil
			}
			return next(ctx)
		}
	}
	return next
}
