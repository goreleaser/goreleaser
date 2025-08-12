package retry

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/caarlos0/log"
)

// Retriable is something that can retry an operation.
type Retriable[T any] interface {
	// Do retries the given function until it succeeds or the maximum number of
	// attempts is reached.
	Do(func() (T, error)) (T, error)
}

// New returns a new Retriable instance.
func New[T any](
	op string,
	max int,
	initialInterval time.Duration,
	maxInterval time.Duration,
	isRetryable func(error) bool,
) Retriable[T] {
	return retry[T]{
		op:              op,
		max:             max,
		initialInterval: initialInterval,
		maxInterval:     maxInterval,
		isRetryable:     isRetryable,
	}
}

type retry[T any] struct {
	op              string
	max             int
	initialInterval time.Duration
	maxInterval     time.Duration
	isRetryable     func(error) bool
}

func (r retry[T]) Do(fn func() (T, error)) (T, error) {
	var result T
	var err error
	for try := 0; try < r.max; try++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		if !r.isRetryable(err) {
			return result, fmt.Errorf("failed to %s after %d tries: %w", r.op, try+1, err)
		}

		if try < r.max-1 {
			exponential := float64(r.initialInterval) * math.Pow(2, float64(try))
			jitter := time.Duration(rand.Float64() * min(exponential, float64(r.maxInterval)))
			log.WithField("try", try+1).
				WithError(err).
				Warnf("failed to %s, will retry after %s", r.op, jitter)
			time.Sleep(jitter)
		}
	}
	return result, fmt.Errorf("failed to %s after %d tries: %w", r.op, r.max, err)
}
