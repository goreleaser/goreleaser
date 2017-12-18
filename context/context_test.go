package context

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/tj/assert"
)

func TestNew(t *testing.T) {
	var ctx = New(config.Project{})
	assert.NotEmpty(t, ctx.Env)
	assert.Equal(t, 4, ctx.Parallelism)
}
