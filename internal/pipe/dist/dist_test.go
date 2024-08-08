package dist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "dist", ctx.Config.Dist)
}

func TestDistDoesNotExist(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, Pipe{}.Run(testctx.NewWithCfg(config.Project{Dist: dist})))
}

func TestPopulatedDistExists(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	f, err := os.Create(filepath.Join(dist, "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(config.Project{Dist: dist})
	require.Error(t, Pipe{}.Run(ctx))
	require.NoError(t, CleanPipe{}.Run(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	_, err = os.Stat(dist)
	require.False(t, os.IsExist(err))
}

func TestEmptyDistExists(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.NewWithCfg(config.Project{Dist: dist})
	require.NoError(t, Pipe{}.Run(ctx))
	_, err := os.Stat(dist)
	require.False(t, os.IsNotExist(err))
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
	require.NotEmpty(t, CleanPipe{}.String())
}

func TestCleanSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, CleanPipe{}.Skip(testctx.New()))
	})
	t.Run("don't skip", func(t *testing.T) {
		require.False(t, CleanPipe{}.Skip(testctx.New(func(ctx *context.Context) {
			ctx.Clean = true
		})))
	})
}

func TestCleanSetDist(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, CleanPipe{}.Run(ctx))
	require.Equal(t, "dist", ctx.Config.Dist)
}
