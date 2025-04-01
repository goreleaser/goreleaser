// Package pipe provides generic errors for pipes to use.
package pipe

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrSnapshotEnabled happens when goreleaser is running in snapshot mode.
// It usually means that publishing and maybe some validations were skipped.
var ErrSnapshotEnabled = Skip("disabled during snapshot mode")

// ErrSkipPublishEnabled happens if --skip=publish is set.
// It means that the part of a Piper that publishes its artifacts was not run.
var ErrSkipPublishEnabled = Skip("publishing is disabled")

// ErrSkipAnnounceEnabled happens if --skip=announce is set.
var ErrSkipAnnounceEnabled = Skip("announcing is disabled")

// ErrSkipSignEnabled happens if --skip=sign is set.
// It means that the part of a Piper that signs some things was not run.
var ErrSkipSignEnabled = Skip("artifact signing is disabled")

// ErrSkipValidateEnabled happens if --skip=validate is set.
// It means that the part of a Piper that validates some things was not run.
var ErrSkipValidateEnabled = Skip("validation is disabled")

// IsSkip returns true if the error is an ErrSkip.
func IsSkip(err error) bool {
	return errors.As(err, &ErrSkip{})
}

// ErrSkip occurs when a pipe is skipped for some reason.
type ErrSkip struct {
	reason string
}

// Error implements the error interface. returns the reason the pipe was skipped.
func (e ErrSkip) Error() string {
	return e.reason
}

// Skip skips this pipe with the given reason.
func Skip(reason string) ErrSkip {
	return ErrSkip{reason: reason}
}

// Skipf skips this pipe with the given reason.
func Skipf(format string, a ...any) ErrSkip {
	return Skip(fmt.Sprintf(format, a...))
}

// SkipMemento remembers previous skip errors so you can return them all at once later.
type SkipMemento struct {
	skips []string
}

// Remember a skip.
func (e *SkipMemento) Remember(err error) {
	if slices.Contains(e.skips, err.Error()) {
		return
	}
	e.skips = append(e.skips, err.Error())
}

// Evaluate return a skip error with all previous skips, or nil if none happened.
func (e *SkipMemento) Evaluate() error {
	if len(e.skips) == 0 {
		return nil
	}
	return Skip(strings.Join(e.skips, ", "))
}

// DetailsOf gets the details of an error, if available.
func DetailsOf(err error) map[string]any {
	if de, ok := err.(errDetailed); ok {
		return de.details
	}
	return map[string]any{}
}

// NewDetailedError makes an error with details, mainly used for logging.
func NewDetailedError(err error, pairs ...any) error {
	details := map[string]any{}
	if len(pairs)%2 != 0 {
		pairs = append(pairs, "missing value")
	}
	for i := 0; i < len(pairs); i += 2 {
		details[fmt.Sprintf("%v", pairs[i])] = pairs[i+1]
	}
	return errDetailed{
		err:     err,
		details: details,
	}
}

type errDetailed struct {
	err     error
	details map[string]any
}

// Error implements error.
func (e errDetailed) Error() string { return e.err.Error() }

// Unwrap implements unwrap.
func (e errDetailed) Unwrap() error { return e.err }
