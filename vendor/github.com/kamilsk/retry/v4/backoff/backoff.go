// Package backoff provides stateless methods of calculating durations based on
// a number of attempts made.
package backoff

import (
	"math"
	"time"
)

// Algorithm defines a function that calculates a time.Duration based on
// the given retry attempt number.
type Algorithm func(attempt uint) time.Duration

// Constant creates an Algorithm that returns the initial duration
// by the all time.
func Constant(duration time.Duration) Algorithm {
	return func(uint) time.Duration {
		return duration
	}
}

// Incremental creates an Algorithm that increments the initial duration
// by the given increment for each attempt.
func Incremental(initial, increment time.Duration) Algorithm {
	return func(attempt uint) time.Duration {
		return initial + (increment * time.Duration(attempt))
	}
}

// Linear creates an Algorithm that linearly multiplies the factor
// duration by the attempt number for each attempt.
func Linear(factor time.Duration) Algorithm {
	return Incremental(0, factor)
}

// Exponential creates an Algorithm that multiplies the factor duration by
// an exponentially increasing factor for each attempt, where the factor is
// calculated as the given base raised to the attempt number.
func Exponential(factor time.Duration, base float64) Algorithm {
	return func(attempt uint) time.Duration {
		return factor * time.Duration(math.Pow(base, float64(attempt)))
	}
}

// BinaryExponential creates an Algorithm that multiplies the factor
// duration by an exponentially increasing factor for each attempt, where the
// factor is calculated as `2` raised to the attempt number (2^attempt).
func BinaryExponential(factor time.Duration) Algorithm {
	return Exponential(factor, 2)
}

// Fibonacci creates an Algorithm that multiplies the factor duration by
// an increasing factor for each attempt, where the factor is the Nth number in
// the Fibonacci sequence.
func Fibonacci(factor time.Duration) Algorithm {
	return func(attempt uint) time.Duration {
		return factor * time.Duration(fibonacciNumber(attempt))
	}
}

// fibonacciNumber calculates the Fibonacci sequence number for the given
// sequence position.
func fibonacciNumber(n uint) uint {
	if n == 0 {
		return 0
	}
	var a, b uint = 0, 1
	for i := uint(1); i < n; i++ {
		a, b = b, a+b
	}
	return b
}
