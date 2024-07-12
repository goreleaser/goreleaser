package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/stretchr/testify/require"
)

// AssertSkipped asserts that a pipe was skipped.
func AssertSkipped(t *testing.T, err error) {
	t.Helper()
	require.ErrorAs(t, err, &pipe.ErrSkip{}, "expected a pipe.ErrSkip but got %v", err)
}
