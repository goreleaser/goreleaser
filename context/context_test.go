package context

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/config"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var ctx = New(config.Project{})
	assert.NotEmpty(t, ctx.Env)
	assert.Equal(t, 4, ctx.Parallelism)
}

func TestNewWithTimeout(t *testing.T) {
	ctx, cancel := NewWithTimeout(config.Project{}, time.Second)
	assert.NotEmpty(t, ctx.Env)
	assert.Equal(t, 4, ctx.Parallelism)
	cancel()
	<-ctx.Done()
	assert.EqualError(t, ctx.Err(), `context canceled`)
}
