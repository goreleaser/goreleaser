package nodesea

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic(t *testing.T) {
	t.Run("writes data with requested perms", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		require.NoError(t, writeFileAtomic(path, []byte("hello"), 0o755))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, []byte("hello"), got)

		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
		}
	})

	t.Run("replaces existing file atomically", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		require.NoError(t, os.WriteFile(path, []byte("original"), 0o644))
		require.NoError(t, writeFileAtomic(path, []byte("replaced"), 0o755))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, []byte("replaced"), got)
	})

	t.Run("leaves no temp file in target dir on success", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.bin")

		require.NoError(t, writeFileAtomic(path, []byte("data"), 0o755))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "out.bin", entries[0].Name())
	})

	t.Run("fails when target dir does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing-subdir", "out.bin")
		require.Error(t, writeFileAtomic(path, []byte("x"), 0o755))
	})
}
