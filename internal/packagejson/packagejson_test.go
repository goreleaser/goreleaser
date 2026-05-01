package packagejson

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	pkg, err := Open("./testdata/package.json")
	require.NoError(t, err)
	require.Equal(t, "example", pkg.Name)
	require.Equal(t, "index.ts", pkg.Module)
	require.Equal(t, "module", pkg.Type)
	require.True(t, pkg.IsBun())
	require.Equal(t, ">=22.20.0", pkg.Engines.NodeRange())
	require.True(t, pkg.Scripts.HasBuild())
	require.Equal(t, "esbuild src/index.ts", pkg.Scripts.Build)
}

func TestOpenOrEmpty(t *testing.T) {
	t.Run("missing file → zero Package", func(t *testing.T) {
		pkg, err := OpenOrEmpty(filepath.Join(t.TempDir(), "missing.json"))
		require.NoError(t, err)
		require.Empty(t, pkg.Name)
		require.False(t, pkg.Scripts.HasBuild())
		require.Empty(t, pkg.Engines.NodeRange())
	})

	t.Run("present file → parsed", func(t *testing.T) {
		pkg, err := OpenOrEmpty("./testdata/package.json")
		require.NoError(t, err)
		require.Equal(t, "example", pkg.Name)
	})
}
