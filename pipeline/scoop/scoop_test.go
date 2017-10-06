package scoop

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipeNoFormats(t *testing.T) {
	var ctx = &context.Context{
		Version: "1.0.0",
		Config:  config.Project{},
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}
