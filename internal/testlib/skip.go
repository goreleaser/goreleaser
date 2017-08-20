package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/stretchr/testify/assert"
)

// AssertSkipped asserts that a pipe was skipped
func AssertSkipped(t *testing.T, err error) {
	_, ok := err.(pipeline.ErrSkip)
	assert.True(t, ok)
}
