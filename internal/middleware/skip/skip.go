// Package skip can skip an entire Action.
package skip

import (
	"fmt"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Skipper defines a method to skip an entire Piper.
type Skipper interface {
	// Skip returns true if the Piper should be skipped.
	Skip(ctx *context.Context) bool
	fmt.Stringer
}

// Skipper defines a method to skip an entire Piper.
type ErrSkipper interface {
	// Skip returns true if the Piper should be skipped.
	Skip(ctx *context.Context) (bool, error)
	fmt.Stringer
}

// Maybe returns an action that skips immediately if the given p is a Skipper
// and its Skip method returns true.
func Maybe(skipper interface{}, next middleware.Action) middleware.Action {
	if skipper, ok := skipper.(Skipper); ok {
		return Maybe(wrapper{skipper}, next)
	}
	if skipper, ok := skipper.(ErrSkipper); ok {
		return func(ctx *context.Context) error {
			skip, err := skipper.Skip(ctx)
			if err != nil {
				return fmt.Errorf("skip %s: %w", skipper.String(), err)
			}
			if skip {
				log.Debugf("skipped %s", skipper.String())
				return nil
			}
			return next(ctx)
		}
	}
	return next
}

var _ ErrSkipper = wrapper{}

type wrapper struct {
	skipper Skipper
}

// String implements SkipperErr
func (w wrapper) String() string {
	return w.skipper.String()
}

// Skip implements SkipperErr
func (w wrapper) Skip(ctx *context.Context) (bool, error) {
	return w.skipper.Skip(ctx), nil
}
