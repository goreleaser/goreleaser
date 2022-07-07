package testlib

import (
	"errors"
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/stretchr/testify/require"
)

// AssertSkipped asserts that a pipe was skipped.
func AssertSkipped(t *testing.T, err error) {
	t.Helper()
	require.True(t, errors.As(err, &pipe.ErrSkip{}), "expected a pipe.ErrSkip but got %v", err)
}
