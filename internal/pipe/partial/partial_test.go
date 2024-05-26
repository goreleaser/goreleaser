package partial

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

var pipe = Pipe{}

func TestString(t *testing.T) {
	require.NotEmpty(t, pipe.String())
}

func TestSkip(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		ctx := testctx.New(testctx.Partial)
		require.False(t, pipe.Skip(ctx))
	})

	t.Run("full", func(t *testing.T) {
		require.True(t, pipe.Skip(testctx.New()))
	})
}

func TestRun(t *testing.T) {
	t.Run("target", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		t.Setenv("GOOS", "windows")
		t.Setenv("GOARCH", "arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("using GGOOS and GGOARCH", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		t.Setenv("GGOOS", "windows")
		t.Setenv("GGOARCH", "arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("using runtime", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		require.NoError(t, pipe.Run(ctx))
		target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		require.Equal(t, target, ctx.PartialTarget)
	})
}
