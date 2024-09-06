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
	t.Run("custom GGOARM", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "arm")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm_6", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOARM", "7")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm_7", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOAMD64", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "amd64")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_amd64_v1", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOAMD64", "v4")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_amd64_v4", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOMIPS", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		for _, mips := range []string{"mips", "mips64", "mipsle", "mips64le"} {
			t.Run(mips, func(t *testing.T) {
				t.Setenv("GGOARCH", mips)
				t.Run("default", func(t *testing.T) {
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips+"_hardfloat", ctx.PartialTarget)
				})
				t.Run("default", func(t *testing.T) {
					t.Setenv("GGOMIPS", "softfloat")
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips+"_softfloat", ctx.PartialTarget)
				})
			})
		}
	})
	t.Run("using runtime", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		require.NoError(t, pipe.Run(ctx))
		target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		// commonly tests will run on either arm64 or amd64.
		if runtime.GOARCH == "amd64" {
			target += "_v1"
		}
		require.Equal(t, target, ctx.PartialTarget)
	})
}
