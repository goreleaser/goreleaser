package nodesea

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func withStubIndex(t *testing.T, entries []indexEntry) {
	t.Helper()
	original := indexFetcher
	indexFetcher = func(_ context.Context) ([]indexEntry, error) {
		return entries, nil
	}
	t.Cleanup(func() { indexFetcher = original })
}

func TestResolveVersion(t *testing.T) {
	ctx := t.Context()
	stub := []indexEntry{
		{Version: "v23.0.0"},
		{Version: "v22.10.0"},
		{Version: "v22.9.0"},
		{Version: "v20.18.0"},
		{Version: "v18.20.4"},
	}

	t.Run("explicit pinned", func(t *testing.T) {
		dir := t.TempDir()
		v, src, err := ResolveVersion(ctx, dir, "22.10.0")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceExplicit, src)
	})

	t.Run("explicit with v prefix", func(t *testing.T) {
		dir := t.TempDir()
		v, _, err := ResolveVersion(ctx, dir, "v22.10.0")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
	})

	t.Run("explicit semver range", func(t *testing.T) {
		withStubIndex(t, stub)
		dir := t.TempDir()
		v, src, err := ResolveVersion(ctx, dir, "^22")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceExplicit, src)
	})

	t.Run("from package.json engines.node", func(t *testing.T) {
		withStubIndex(t, stub)
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"),
			[]byte(`{"engines":{"node":">=22 <23"}}`), 0o644))
		v, src, err := ResolveVersion(ctx, dir, "")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceEnginesNode, src)
	})

	t.Run("from .nvmrc", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v22.10.0\n"), 0o644))
		v, src, err := ResolveVersion(ctx, dir, "")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceNvmrc, src)
	})

	t.Run("from .node-version", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".node-version"), []byte("22.10.0\n"), 0o644))
		v, src, err := ResolveVersion(ctx, dir, "")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceNodeVersion, src)
	})

	t.Run("nothing set", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := ResolveVersion(ctx, dir, "")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrNoVersion))
	})

	t.Run("explicit takes precedence over files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("18.0.0\n"), 0o644))
		v, src, err := ResolveVersion(ctx, dir, "22.10.0")
		require.NoError(t, err)
		require.Equal(t, "v22.10.0", v)
		require.Equal(t, VersionSourceExplicit, src)
	})

	t.Run("range with no match", func(t *testing.T) {
		withStubIndex(t, stub)
		dir := t.TempDir()
		_, _, err := ResolveVersion(ctx, dir, "^99")
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
