package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/pipe"
)

func TestAssertSkipped(t *testing.T) {
	AssertSkipped(t, pipe.Skip("skip"))
}
