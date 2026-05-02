package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractFromTarGz_AtomicOnFailure(t *testing.T) {
	const entry = "node-v22.10.0-linux-x64/bin/node"

	// Build a valid tarball with a 4 KiB payload, then truncate the
	// gzipped bytes so io.Copy hits unexpected EOF mid-extract.
	payload := bytes.Repeat([]byte{'A'}, 4096)
	archive := tarGz(t, entry, payload)
	truncated := archive[:len(archive)/2]

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "truncated.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, truncated, 0o644))

	dst := filepath.Join(dir, "node")
	err := extractFromTarGz(archivePath, entry, dst)
	require.Error(t, err)
	_, statErr := os.Stat(dst)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	leftovers, err := filepath.Glob(filepath.Join(dir, ".extract-*"))
	require.NoError(t, err)
	require.Empty(t, leftovers, "tempfile not cleaned up")
}

func TestExtractFromTarGz_HappyPath(t *testing.T) {
	const entry = "node-v22.10.0-linux-x64/bin/node"
	payload := []byte("fake node binary")
	archive := tarGz(t, entry, payload)

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "ok.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, archive, 0o644))

	dst := filepath.Join(dir, "node")
	require.NoError(t, extractFromTarGz(archivePath, entry, dst))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestTarget(t *testing.T) {
	linux := mustParseTarget(t, "linux-x64")
	win := mustParseTarget(t, "win-x64")
	require.Equal(t, "linux", linux.Goos())
	require.Equal(t, "windows", win.Goos())
	require.Equal(t, "amd64", linux.Goarch())
}

// tarGz builds a single-entry gzipped tarball for tests.
func tarGz(t *testing.T, entry string, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: entry,
		Mode: 0o755,
		Size: int64(len(payload)),
	}))
	_, err := tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}
