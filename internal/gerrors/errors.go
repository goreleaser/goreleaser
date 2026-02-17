// Package gerrors provides error handling for GoReleaser.
package gerrors

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
)

// Option changes things in an [ErrDetailed].
type Option func(*ErrDetailed)

// WithExit sets the exit code in an [ErrDetailed].
func WithExit(exit int) Option {
	return func(ed *ErrDetailed) {
		ed.exit = exit
	}
}

// WithMessage adds a message to an [ErrDetailed].
func WithMessage(message string) Option {
	return func(ed *ErrDetailed) {
		ed.messages = append(ed.messages, message)
	}
}

// WithOutput sets the output in an [ErrDetailed].
func WithOutput(output string) Option {
	return func(ed *ErrDetailed) {
		ed.output = output
	}
}

// WithDetails adds details to an [ErrDetailed].
//
// Details are key-value pairs, so the number of arguments should be even.
func WithDetails(pairs ...any) Option {
	return func(ed *ErrDetailed) {
		if len(pairs)%2 != 0 {
			pairs = append(pairs, "missing value")
		}
		ed.details = pairs
	}
}

// Wrap makes an error with details, mainly used for logging.
func Wrap(err error, opts ...Option) error {
	result := ErrDetailed{
		err:  err,
		exit: 1,
	}

	for _, opt := range opts {
		opt(&result)
	}

	if de, ok := errors.AsType[ErrDetailed](err); ok {
		result.details = append(de.details, result.details...)
		result.messages = append(result.messages, de.messages...)
		result.output = cmp.Or(result.output, de.output)
	}

	return result
}

// ErrDetailed is an error with details, mainly used for logging.
type ErrDetailed struct {
	err      error
	exit     int
	output   string
	messages []string
	details  []any
}

// Details returns the details of an [ErrDetailed] as a sequence of key-value
// pairs.
func (e ErrDetailed) Details() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for i := 0; i < len(e.details); i += 2 {
			key, ok := e.details[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", e.details[i])
			}
			if !yield(key, e.details[i+1]) {
				break
			}
		}
	}
}

// Error implements error.
func (e ErrDetailed) Error() string { return e.err.Error() }

// Unwrap implements unwrap.
func (e ErrDetailed) Unwrap() error { return e.err }

// Exit gets the exit code of an error, if available.
func (e ErrDetailed) Exit() int { return e.exit }

// Messages returns the messages of an [ErrDetailed].
func (e ErrDetailed) Messages() []string { return e.messages }

// Output returns the output of an [ErrDetailed].
func (e ErrDetailed) Output() string { return e.output }
