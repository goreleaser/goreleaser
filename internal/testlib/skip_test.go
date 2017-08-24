package testlib

import (
	"testing"

	"github.com/goreleaser/goreleaser/pipeline"
)

func TestAssertSkipped(t *testing.T) {
	AssertSkipped(t, pipeline.Skip("skip"))
}
