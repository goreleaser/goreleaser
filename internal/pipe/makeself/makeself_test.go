package makeself

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.Equal(t, "makeself packages", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Makeself))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Makeselfs: []config.Makeself{{}},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no makeselfs", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.True(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Makeselfs: []config.Makeself{
			{},
			{
				ID:       "custom",
				Name:     "custom",
				Filename: "custom_{{.Os}}_{{.Arch}}.bin",
				Goos:     []string{"freebsd"},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Makeselfs, 2)

	m1 := ctx.Config.Makeselfs[0]
	require.Equal(t, "default", m1.ID)
	require.NotEmpty(t, m1.Name)
	require.Equal(t, defaultNameTemplate, m1.Filename)
	require.Len(t, m1.Goos, 2)

	m2 := ctx.Config.Makeselfs[1]
	require.Equal(t, "custom", m2.ID)
	require.Equal(t, "custom", m2.Name)
	require.Equal(t, "custom_{{.Os}}_{{.Arch}}.bin", m2.Filename)
	require.Len(t, m2.Goos, 1)
}
