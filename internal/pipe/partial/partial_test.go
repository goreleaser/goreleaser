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
		t.Setenv("TARGET", "windows_arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("no target", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
		}, testctx.Partial)
		require.Error(t, pipe.Run(ctx))
	})
	t.Run("using GGOOS and GGOARCH", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "windows")
		t.Setenv("GGOARCH", "arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("custom GGOARM", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "arm")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOARM", "7")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm_7", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOARM64", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "arm64")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm64", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOARM64", "v9.0")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_arm64_v9.0", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOAMD64", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "amd64")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_amd64", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOAMD64", "v4")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_amd64_v4", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOMIPS", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		for _, mips := range []string{"mips", "mips64", "mipsle", "mips64le"} {
			t.Run(mips, func(t *testing.T) {
				t.Setenv("GGOARCH", mips)
				t.Run("default", func(t *testing.T) {
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips, ctx.PartialTarget)
				})
				t.Run("default", func(t *testing.T) {
					t.Setenv("GGOMIPS", "softfloat")
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips+"_softfloat", ctx.PartialTarget)
				})
			})
		}
	})
	t.Run("custom GGO386", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "386")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_386", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGO386", "softfloat")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_386_softfloat", ctx.PartialTarget)
		})
	})
	t.Run("custom GGOPPC64", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "ppc64")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_ppc64", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGOPPC64", "power9")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_ppc64_power9", ctx.PartialTarget)
		})
	})
	t.Run("custom GGORISCV64", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "riscv64")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_riscv64", ctx.PartialTarget)
		})
		t.Run("default", func(t *testing.T) {
			t.Setenv("GGORISCV64", "rva22u64")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_riscv64_rva22u64", ctx.PartialTarget)
		})
	})
	t.Run("using runtime", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)
		require.NoError(t, pipe.Run(ctx))
		target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		require.Equal(t, target, ctx.PartialTarget)
	})

	t.Run("using runtime with other languages", func(t *testing.T) {
		t.Setenv("GGOOS", "darwin")
		t.Setenv("GGOARCH", "amd64")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
			Builds: []config.Build{{
				Builder: "rust",
				Targets: []string{
					"x86_64-unknown-linux-gnu",
					"x86_64-apple-darwin",
					"x86_64-pc-windows-gnu",
					"aarch64-unknown-linux-gnu",
					"aarch64-apple-darwin",
				},
			}},
		}, testctx.Partial)
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "x86_64-apple-darwin", ctx.PartialTarget)
	})

	t.Run("using runtime with other languages no match", func(t *testing.T) {
		t.Setenv("GGOOS", "darwin")
		t.Setenv("GGOARCH", "amd64")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: "dist",
			Builds: []config.Build{{
				Builder: "rust",
				Targets: []string{
					"x86_64-unknown-linux-gnu",
					"aarch64-unknown-linux-gnu",
				},
			}},
		}, testctx.Partial)
		require.Error(t, pipe.Run(ctx))
		require.Empty(t, ctx.PartialTarget)
	})
}
