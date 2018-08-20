package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipeline"
)

func TestAssertSkipped(t *testing.T) {
	AssertSkipped(t, pipeline.Skip("skip"))
}
