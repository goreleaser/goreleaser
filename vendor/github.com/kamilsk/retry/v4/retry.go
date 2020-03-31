// Package retry provides the most advanced interruptible mechanism
// to perform actions repetitively until successful.
package retry

import "sync/atomic"

// Retry takes an action and performs it, repetitively, until successful.
// When it is done it releases resources associated with the Breaker.
//
// Optionally, strategies may be passed that assess whether or not an attempt
// should be made.
//
// Deprecated: will be replaced by Do function (current Try).
// TODO:v5 will be removed
func Retry(
	breaker BreakCloser,
	action func(attempt uint) error,
	strategies ...func(attempt uint, err error) bool,
) error {
	err := retry(breaker, action, strategies...)
	breaker.Close()
	return err
}

// Try takes an action and performs it, repetitively, until successful.
//
// Optionally, strategies may be passed that assess whether or not an attempt
// should be made.
//
// TODO:v5 will be renamed to Do
func Try(
	breaker Breaker,
	action func(attempt uint) error,
	strategies ...func(attempt uint, err error) bool,
) error {
	return retry(breaker, action, strategies...)
}

func retry(
	breaker Breaker,
	action func(attempt uint) error,
	strategies ...func(attempt uint, err error) bool,
) error {
	var interrupted uint32
	done := make(chan result, 1)

	go func(breaker *uint32) {
		var err error

		defer func() {
			done <- result{err, recover()}
			close(done)
		}()

		for attempt := uint(0); shouldAttempt(breaker, attempt, err, strategies...); attempt++ {
			err = action(attempt)
		}
	}(&interrupted)

	select {
	case <-breaker.Done():
		atomic.StoreUint32(&interrupted, 1)
		return Interrupted
	case err := <-done:
		if _, is := IsRecovered(err); is {
			return err
		}
		return err.error
	}
}

// shouldAttempt evaluates the provided strategies with the given attempt to
// determine if the Retry loop should make another attempt.
func shouldAttempt(breaker *uint32, attempt uint, err error, strategies ...func(uint, error) bool) bool {
	should := attempt == 0 || err != nil

	for i, repeat := 0, len(strategies); should && i < repeat; i++ {
		should = should && strategies[i](attempt, err)
	}

	return should && atomic.LoadUint32(breaker) == 0
}
