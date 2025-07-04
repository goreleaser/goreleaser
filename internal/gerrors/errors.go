// Package gerrors provides error handling for GoReleaser.
package gerrors

import (
	"errors"
	"fmt"
	"maps"
)

// DetailsOf gets the details of an error, if available.
func DetailsOf(err error) map[string]any {
	var de errDetailed
	if errors.As(err, &de) {
		return de.details
	}
	return map[string]any{}
}

// ExitOf gets the exit code of an error, if available.
func ExitOf(err error) int {
	var de errDetailed
	if errors.As(err, &de) {
		return de.exit
	}
	return 1
}

// MessageOf gets the message of an error, if available.
func MessageOf(err error) string {
	var de errDetailed
	if errors.As(err, &de) {
		return de.message
	}
	return ""
}

// WrapExit makes an error with details and an exit code, mainly used for logging.
func WrapExit(err error, message string, exit int, pairs ...any) error {
	details := map[string]any{}
	if len(pairs)%2 != 0 {
		pairs = append(pairs, "missing value")
	}
	for i := 0; i < len(pairs); i += 2 {
		details[fmt.Sprintf("%v", pairs[i])] = pairs[i+1]
	}
	if dets := DetailsOf(err); len(dets) > 0 {
		maps.Copy(details, dets)
	}
	return errDetailed{
		err:     err,
		details: details,
		exit:    exit,
		message: message,
	}
}

// Wrap makes an error with details, mainly used for logging.
func Wrap(err error, message string, pairs ...any) error {
	return WrapExit(err, message, 1, pairs...)
}

type errDetailed struct {
	err     error
	exit    int
	message string
	details map[string]any
}

// Error implements error.
func (e errDetailed) Error() string { return e.err.Error() }

// Unwrap implements unwrap.
func (e errDetailed) Unwrap() error { return e.err }
