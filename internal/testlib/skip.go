package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/stretchr/testify/assert"
)

// AssertSkipped asserts that a pipe was skipped
func AssertSkipped(t *testing.T, err error) {
	_, ok := err.(pipe.ErrSkip)
	assert.True(t, ok, "expected a pipe.ErrSkip but got %v", err)
}
