package context

import (
	"os"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	require.NoError(t, os.Setenv("FOO", "NOT BAR"))
	require.NoError(t, os.Setenv("BAR", "1"))
	var ctx = New(config.Project{
		Env: []string{
			"FOO=BAR",
		},
	})
	require.Equal(t, "BAR", ctx.Env["FOO"])
	require.Equal(t, "1", ctx.Env["BAR"])
	require.Equal(t, 4, ctx.Parallelism)
}

func TestNewWithTimeout(t *testing.T) {
	ctx, cancel := NewWithTimeout(config.Project{}, time.Second)
	require.NotEmpty(t, ctx.Env)
	require.Equal(t, 4, ctx.Parallelism)
	cancel()
	<-ctx.Done()
	require.EqualError(t, ctx.Err(), `context canceled`)
}
