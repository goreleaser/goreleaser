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
		ctx := testctx.Wrap(t.Context(), testctx.Partial)
		require.False(t, pipe.Skip(ctx))
	})

	t.Run("full", func(t *testing.T) {
		require.True(t, pipe.Skip(testctx.Wrap(t.Context())))
	})
}

func TestRun(t *testing.T) {
	t.Run("target", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: "dist",
		}, testctx.Partial)

		t.Setenv("TARGET", "windows_arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("no target", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: "dist",
		}, testctx.Partial)

		require.Error(t, pipe.Run(ctx))
	})
	t.Run("using GGOOS and GGOARCH", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)

		t.Setenv("GGOOS", "windows")
		t.Setenv("GGOARCH", "arm64")
		require.NoError(t, pipe.Run(ctx))
		require.Equal(t, "windows_arm64", ctx.PartialTarget)
	})
	t.Run("custom GGOARM", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)

		t.Setenv("GGOOS", "linux")
		for _, mips := range []string{"mips", "mipsle"} {
			t.Run(mips, func(t *testing.T) {
				t.Setenv("GGOARCH", mips)
				t.Run("default", func(t *testing.T) {
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips, ctx.PartialTarget)
				})
				t.Run("with value", func(t *testing.T) {
					t.Setenv("GGOMIPS", "softfloat")
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips+"_softfloat", ctx.PartialTarget)
				})
			})
		}
	})
	t.Run("custom GGOMIPS64", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)

		t.Setenv("GGOOS", "linux")
		for _, mips := range []string{"mips64", "mips64le"} {
			t.Run(mips, func(t *testing.T) {
				t.Setenv("GGOARCH", mips)
				t.Run("default", func(t *testing.T) {
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips, ctx.PartialTarget)
				})
				t.Run("with value", func(t *testing.T) {
					t.Setenv("GGOMIPS64", "softfloat")
					require.NoError(t, pipe.Run(ctx))
					require.Equal(t, "linux_"+mips+"_softfloat", ctx.PartialTarget)
				})
			})
		}
	})
	t.Run("custom GGO386", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	t.Run("GOMIPS64 fallback for mips64", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)

		t.Setenv("GGOOS", "linux")
		for _, mips := range []string{"mips64", "mips64le"} {
			t.Run(mips, func(t *testing.T) {
				t.Setenv("GGOARCH", mips)
				t.Setenv("GOMIPS64", "softfloat")
				require.NoError(t, pipe.Run(ctx))
				require.Equal(t, "linux_"+mips+"_softfloat", ctx.PartialTarget)
			})
		}
	})
	t.Run("custom GGOPPC64", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	t.Run("custom GGOPPC64 with ppc64le", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:   "dist",
			Builds: []config.Build{{Builder: "go"}},
		}, testctx.Partial)

		t.Setenv("GGOOS", "linux")
		t.Setenv("GGOARCH", "ppc64le")
		t.Run("default", func(t *testing.T) {
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_ppc64le", ctx.PartialTarget)
		})
		t.Run("with value", func(t *testing.T) {
			t.Setenv("GGOPPC64", "power9")
			require.NoError(t, pipe.Run(ctx))
			require.Equal(t, "linux_ppc64le_power9", ctx.PartialTarget)
		})
	})
	t.Run("custom GGORISCV64", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

	t.Run("using runtime with node and bun targets", func(t *testing.T) {
		for name, tt := range map[string]struct {
			builder  string
			goos     string
			goarch   string
			targets  []string
			expected string
		}{
			"node darwin arm64": {
				builder:  "node",
				goos:     "darwin",
				goarch:   "arm64",
				targets:  []string{"linux-x64", "darwin-arm64"},
				expected: "darwin-arm64",
			},
			"node darwin amd64": {
				builder:  "node",
				goos:     "darwin",
				goarch:   "amd64",
				targets:  []string{"linux-x64", "darwin-x64"},
				expected: "darwin-x64",
			},
			"node linux arm64": {
				builder:  "node",
				goos:     "linux",
				goarch:   "arm64",
				targets:  []string{"darwin-x64", "linux-arm64"},
				expected: "linux-arm64",
			},
			"node linux amd64": {
				builder:  "node",
				goos:     "linux",
				goarch:   "amd64",
				targets:  []string{"darwin-arm64", "linux-x64"},
				expected: "linux-x64",
			},
			"node windows arm64": {
				builder:  "node",
				goos:     "windows",
				goarch:   "arm64",
				targets:  []string{"linux-x64", "win-arm64"},
				expected: "win-arm64",
			},
			"node windows amd64": {
				builder:  "node",
				goos:     "windows",
				goarch:   "amd64",
				targets:  []string{"linux-x64", "win-x64"},
				expected: "win-x64",
			},
			"bun darwin arm64": {
				builder:  "bun",
				goos:     "darwin",
				goarch:   "arm64",
				targets:  []string{"linux-x64-modern", "darwin-arm64"},
				expected: "darwin-arm64",
			},
			"bun darwin amd64": {
				builder:  "bun",
				goos:     "darwin",
				goarch:   "amd64",
				targets:  []string{"linux-arm64", "darwin-x64"},
				expected: "darwin-x64",
			},
			"bun linux amd64": {
				builder:  "bun",
				goos:     "linux",
				goarch:   "amd64",
				targets:  []string{"darwin-arm64", "linux-x64-modern"},
				expected: "linux-x64-modern",
			},
			"bun linux arm64": {
				builder:  "bun",
				goos:     "linux",
				goarch:   "arm64",
				targets:  []string{"darwin-arm64", "linux-arm64"},
				expected: "linux-arm64",
			},
			"bun windows amd64": {
				builder:  "bun",
				goos:     "windows",
				goarch:   "amd64",
				targets:  []string{"darwin-arm64", "windows-x64-modern"},
				expected: "windows-x64-modern",
			},
		} {
			t.Run(name, func(t *testing.T) {
				t.Setenv("GGOOS", tt.goos)
				t.Setenv("GGOARCH", tt.goarch)
				ctx := testctx.WrapWithCfg(t.Context(), config.Project{
					Dist: "dist",
					Builds: []config.Build{{
						Builder: tt.builder,
						Targets: tt.targets,
					}},
				}, testctx.Partial)

				require.NoError(t, pipe.Run(ctx))
				require.Equal(t, tt.expected, ctx.PartialTarget)
			})
		}
	})

	t.Run("using runtime with other languages preserves match across unmatched builds", func(t *testing.T) {
		t.Setenv("GGOOS", "darwin")
		t.Setenv("GGOARCH", "arm64")

		matching := config.Build{
			Builder: "rust",
			Targets: []string{
				"aarch64-apple-darwin",
			},
		}
		unmatched := config.Build{
			Builder: "rust",
			Targets: []string{
				"aarch64-unknown-linux-gnu",
			},
		}

		for name, builds := range map[string][]config.Build{
			"matching first":  {matching, unmatched},
			"matching second": {unmatched, matching},
		} {
			t.Run(name, func(t *testing.T) {
				ctx := testctx.WrapWithCfg(t.Context(), config.Project{
					Dist:   "dist",
					Builds: builds,
				}, testctx.Partial)
				require.NoError(t, pipe.Run(ctx))
				require.Equal(t, "aarch64-apple-darwin", ctx.PartialTarget)
			})
		}
	})

	t.Run("using runtime with other languages no match", func(t *testing.T) {
		t.Setenv("GGOOS", "darwin")
		t.Setenv("GGOARCH", "amd64")
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
