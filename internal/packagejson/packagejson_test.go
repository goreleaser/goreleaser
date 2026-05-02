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
	require.Equal(t, ">=22.20.0", pkg.Engines["node"])
	require.Equal(t, "esbuild src/index.ts", pkg.Scripts["build"])
}

func TestOpenOrEmpty(t *testing.T) {
	t.Run("missing file → zero Package", func(t *testing.T) {
		pkg, err := OpenOrEmpty(filepath.Join(t.TempDir(), "missing.json"))
		require.NoError(t, err)
		require.Empty(t, pkg.Name)
		require.Empty(t, pkg.Scripts)
		require.Empty(t, pkg.Engines)
	})

	t.Run("present file → parsed", func(t *testing.T) {
		pkg, err := OpenOrEmpty("./testdata/package.json")
		require.NoError(t, err)
		require.Equal(t, "example", pkg.Name)
	})
}
