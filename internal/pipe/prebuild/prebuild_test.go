package prebuild

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("good", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env:    []string{"FOO=bar"},
			Builds: []config.Build{{Main: "{{ .Env.FOO }}"}},
		})
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "bar", ctx.Config.Builds[0].Main)
	})

	t.Run("empty", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env:    []string{"FOO="},
			Builds: []config.Build{{Main: "{{ .Env.FOO }}"}},
		})
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, ".", ctx.Config.Builds[0].Main)
	})

	t.Run("bad", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Builds: []config.Build{{Main: "{{ .Env.FOO }}"}},
		})
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("leaves node mains for target-aware rendering", func(t *testing.T) {
		for name, main := range map[string]string{
			"literal":      "index.js",
			"global":       "build/{{ .ProjectName }}.js",
			"target":       "build/{{ .Target }}.js",
			"os":           "build/{{ .Os }}.js",
			"architecture": "build/{{ .Arch }}.js",
		} {
			t.Run(name, func(t *testing.T) {
				ctx := testctx.WrapWithCfg(t.Context(), config.Project{
					ProjectName: "proj",
					Builds: []config.Build{{
						Builder: "node",
						Main:    main,
					}},
				})

				require.NoError(t, Pipe{}.Run(ctx))
				require.Equal(t, main, ctx.Config.Builds[0].Main)
			})
		}
	})
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
