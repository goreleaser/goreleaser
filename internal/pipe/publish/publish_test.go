package publish

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublish(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{Disable: "true"},
	}, testctx.WithTokenType(context.TokenTypeGitHub))
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.SkipPublish)
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.New()))
	})
}
