// Package gerrors provides error handling for GoReleaser.
package gerrors

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
)

type Option func(*ErrDetailed)

func WithExit(exit int) Option {
	return func(ed *ErrDetailed) {
		ed.exit = exit
	}
}

func WithMessage(message string) Option {
	return func(ed *ErrDetailed) {
		ed.messages = append(ed.messages, message)
	}
}

func WithOutput(output string) Option {
	return func(ed *ErrDetailed) {
		ed.output = output
	}
}

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

type ErrDetailed struct {
	err      error
	exit     int
	output   string
	messages []string
	details  []any
}

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
func (e ErrDetailed) Exit() int {
	return e.exit
}

func (e ErrDetailed) Messages() []string {
	return e.messages
}

func (e ErrDetailed) Output() string {
	return e.output
}
