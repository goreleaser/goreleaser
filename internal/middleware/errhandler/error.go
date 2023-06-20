package errhandler

import (
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/middleware"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/hashicorp/go-multierror"
)

// Handle handles an action error, ignoring and logging pipe skipped
// errors.
func Handle(action middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		err := action(ctx)
		if err == nil {
			return nil
		}
		if pipe.IsSkip(err) {
			log.WithField("reason", err.Error()).Warn("pipe skipped")
			return nil
		}
		return err
	}
}

// Memo is a handler that memorizes errors, so you can grab them all in the end
// instead of returning each of them.
type Memo struct {
	err error
}

// Error returns the underlying error.
func (m *Memo) Error() error {
	return m.err
}

// Wrap the given action, memorizing its errors.
// The resulting action will always return a nil error.
func (m *Memo) Wrap(action middleware.Action) middleware.Action {
	return func(ctx *context.Context) error {
		err := action(ctx)
		if err == nil {
			return nil
		}
		m.Memorize(err)
		return nil
	}
}

func (m *Memo) Memorize(err error) {
	if pipe.IsSkip(err) {
		log.WithField("reason", err.Error()).Warn("pipe skipped")
		return
	}
	m.err = multierror.Append(m.err, err)
}
