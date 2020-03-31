package retry

import "context"

// Action defines a callable function that package retry can handle.
type Action func(attempt uint) error

// InterruptibleAction defines a callable function that package retry can handle.
//
// Deprecated: use Action instead, it will be extended.
// TODO:v5 will be removed
type InterruptibleAction func(ctx context.Context, attempt uint) error

// A Breaker carries a cancellation signal to break an action execution.
//
// It is a subset of context.Context and github.com/kamilsk/breaker.Breaker.
type Breaker interface {
	// Done returns a channel that's closed when a cancellation signal occurred.
	Done() <-chan struct{}
}

// A BreakCloser carries a cancellation signal to break an action execution
// and can release resources associated with it.
//
// It is a subset of github.com/kamilsk/breaker.Breaker.
//
// Deprecated: use Breaker instead, it will be extended.
// TODO:v5 will be removed
type BreakCloser interface {
	Breaker
	// Close closes the Done channel and releases resources associated with it.
	Close()
}

// How is an alias for batch of Strategies.
//
//  how := retry.How{
//  	strategy.Limit(3),
//  }
//
type How []func(attempt uint, err error) bool
