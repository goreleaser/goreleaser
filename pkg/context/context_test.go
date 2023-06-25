package context

import (
	"runtime"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Setenv("FOO", "NOT BAR")
	t.Setenv("BAR", "1")
	ctx := New(config.Project{
		Env: []string{
			"FOO=BAR",
		},
	})
	require.Equal(t, "BAR", ctx.Env["FOO"])
	require.Equal(t, "1", ctx.Env["BAR"])
	require.Equal(t, 4, ctx.Parallelism)
	require.Equal(t, runtime.GOOS, ctx.Runtime.Goos)
	require.Equal(t, runtime.GOARCH, ctx.Runtime.Goarch)
}

func TestNewWithTimeout(t *testing.T) {
	ctx, cancel := NewWithTimeout(config.Project{}, time.Second)
	require.NotEmpty(t, ctx.Env)
	require.Equal(t, 4, ctx.Parallelism)
	cancel()
	<-ctx.Done()
	require.EqualError(t, ctx.Err(), `context canceled`)
}

func TestToEnv(t *testing.T) {
	require.Equal(t, Env{"FOO": "BAR"}, ToEnv([]string{"=nope", "FOO=BAR"}))
	require.Equal(t, Env{"FOO": "BAR"}, ToEnv([]string{"nope", "FOO=BAR"}))
	require.Equal(t, Env{"FOO": "BAR", "nope": ""}, ToEnv([]string{"nope=", "FOO=BAR"}))
}
