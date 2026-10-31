package gio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSamePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	other := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(file, []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(other, []byte("a"), 0o644))
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(file, link); err != nil {
		t.Logf("skipping the link case: %v", err)
		link = ""
	}

	require.True(t, SamePath(file, file))
	require.True(t, SamePath(file, filepath.Join(dir, "sub", "..", "a.txt")))
	require.False(t, SamePath(file, other))
	require.False(t, SamePath(file, ""))
	require.False(t, SamePath("", ""))
	if link != "" {
		require.True(t, SamePath(file, link))
	}
}
