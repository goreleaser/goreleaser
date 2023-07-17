package gio

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChtimes(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	require.NoError(t, Chtimes(path, fmt.Sprintf("%d", modTime.Unix())))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(stat.ModTime()))
}

func TestChtimesFileDoesNotExist(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()
	path := filepath.Join(t.TempDir(), "file")

	require.ErrorIs(t, Chtimes(path, fmt.Sprintf("%d", modTime.Unix())), os.ErrNotExist)
}

func TestChtimesInvalidTS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	require.ErrorIs(t, Chtimes(path, "fake"), strconv.ErrSyntax)
}

func TestChtimesEmpty(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, nil, 0o644))

	require.NoError(t, Chtimes(path, ""))

	stat, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, modTime.Equal(stat.ModTime()))
}
