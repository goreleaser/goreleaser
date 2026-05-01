package nodesea

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersion(t *testing.T) {
	t.Run("from package.json engines.node pinned", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":"22.20.0"}}`), 0o644))
		v, src, err := ResolveVersion(dir)
		require.NoError(t, err)
		require.Equal(t, "v22.20.0", v)
		require.Equal(t, VersionSourceEnginesNode, src)
	})

	t.Run("from package.json engines.node range", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":">=22 <23"}}`), 0o644))
		v, src, err := ResolveVersion(dir)
		require.NoError(t, err)
		// Resolved version comes from the embedded release index; just
		// assert it is a v22.x release.
		require.True(t, strings.HasPrefix(v, "v22."), "got %q", v)
		require.Equal(t, VersionSourceEnginesNode, src)
	})

	t.Run("from .nvmrc", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v22.20.0\n"), 0o644))
		v, src, err := ResolveVersion(dir)
		require.NoError(t, err)
		require.Equal(t, "v22.20.0", v)
		require.Equal(t, VersionSourceNvmrc, src)
	})

	t.Run("from .node-version", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".node-version"), []byte("22.20.0\n"), 0o644))
		v, src, err := ResolveVersion(dir)
		require.NoError(t, err)
		require.Equal(t, "v22.20.0", v)
		require.Equal(t, VersionSourceNodeVersion, src)
	})

	t.Run("engines.node takes precedence over .nvmrc", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":"22.20.0"}}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("18.0.0\n"), 0o644))
		v, src, err := ResolveVersion(dir)
		require.NoError(t, err)
		require.Equal(t, "v22.20.0", v)
		require.Equal(t, VersionSourceEnginesNode, src)
	})

	t.Run("nothing set", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := ResolveVersion(dir)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrNoVersion))
	})

	t.Run("range with no match", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":"^99"}}`), 0o644))
		_, _, err := ResolveVersion(dir)
		require.Error(t, err)
	})
}

func TestReadVersionFile(t *testing.T) {
	dir := t.TempDir()

	v, err := readVersionFile(filepath.Join(dir, "missing"))
	require.NoError(t, err)
	require.Empty(t, v)

	path := filepath.Join(dir, ".nvmrc")
	require.NoError(t, os.WriteFile(path, []byte("# comment\n\n22.10.0\n"), 0o644))
	v, err = readVersionFile(path)
	require.NoError(t, err)
	require.Equal(t, "22.10.0", v)
}
