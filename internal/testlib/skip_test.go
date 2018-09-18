package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"
)

func TestAssertSkipped(t *testing.T) {
	AssertSkipped(t, pipe.Skip("skip"))
}
