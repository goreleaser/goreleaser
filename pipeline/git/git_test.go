package git

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestValidVersion(t *testing.T) {
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.NotEmpty(ctx.Git.CurrentTag)
	assert.NotEmpty(ctx.Git.PreviousTag)
	assert.NotEmpty(ctx.Git.Diff)
}
