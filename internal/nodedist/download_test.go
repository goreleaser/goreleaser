package nodedist

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestTarget(t *testing.T) {
	require.Equal(t, "node-v22.10.0-linux-x64.tar.gz", Target("linux-x64").ArchiveName("v22.10.0"))
	require.Equal(t, "win-x64/node.exe", Target("win-x64").ArchiveName("v22.10.0"))
	require.Equal(t, "node", Target("linux-x64").HostBinaryName())
	require.Equal(t, "node.exe", Target("win-x64").HostBinaryName())
	require.Equal(t, "linux", Target("linux-x64").Goos())
	require.Equal(t, "windows", Target("win-x64").Goos())
	require.Equal(t, "amd64", Target("linux-x64").Goarch())
	require.Equal(t, "arm", Target("linux-armv7l").Goarch())
	require.True(t, Target("win-x64").IsWindows())
	require.False(t, Target("linux-x64").IsWindows())
}

func TestDownload_Linux(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")
	payload := []byte("fake node binary contents")
	archive := FakeArchive(t, version, target, payload)
	archName := target.ArchiveName(version)

	server := NewServer(t, map[string][]byte{
		"/" + version + "/" + archName:    archive,
		"/" + version + "/SHASUMS256.txt": []byte(SHALine(archive, archName)),
	})
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	SetBaseURL(t, server.URL)

	cache := t.TempDir()
	hostPath, err := Download(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.FileExists(t, hostPath)
	got, err := os.ReadFile(hostPath)
	require.NoError(t, err)
	require.Equal(t, payload, got)

	// Second call hits cache, should not re-fetch.
	server.Close()
	hostPath2, err := Download(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.Equal(t, hostPath, hostPath2)
}

func TestDownload_Windows(t *testing.T) {
	const version = "v22.10.0"
	target := Target("win-x64")
	payload := []byte("fake node.exe contents")
	archName := target.ArchiveName(version)

	server := NewServer(t, map[string][]byte{
		"/" + version + "/win-x64/node.exe": payload,
		"/" + version + "/SHASUMS256.txt":   []byte(SHALine(payload, archName)),
	})
	SetBaseURL(t, server.URL)

	cache := t.TempDir()
	hostPath, err := Download(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.FileExists(t, hostPath)
	require.Equal(t, "node.exe", filepath.Base(hostPath))
	got, err := os.ReadFile(hostPath)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestDownload_BadSHA(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")
	archive := FakeArchive(t, version, target, []byte("payload"))
	archName := target.ArchiveName(version)
	shaLine := "deadbeef  " + archName + "\n"

	server := NewServer(t, map[string][]byte{
		"/" + version + "/" + archName:    archive,
		"/" + version + "/SHASUMS256.txt": []byte(shaLine),
	})
	SetBaseURL(t, server.URL)

	_, err := Download(t.Context(), t.TempDir(), version, target)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA-256 mismatch")
}

func TestCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/somewhere")
	dir, err := CacheDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join("/tmp/somewhere", "goreleaser", "node"), dir)
}

func TestExtractNodeFromTarGz_AtomicOnFailure(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")

	// Build a valid tarball with a 4 KiB payload, then truncate the
	// gzipped bytes so that io.Copy hits unexpected EOF mid-extract.
	payload := bytes.Repeat([]byte{'A'}, 4096)
	archive := FakeArchive(t, version, target, payload)
	truncated := archive[:len(archive)/2]

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "truncated.tar.gz")
	require.NoError(t, os.WriteFile(archivePath, truncated, 0o644))

	dst := filepath.Join(dir, "node")
	err := extractNodeFromTarGz(archivePath, version, target, dst)
	require.Error(t, err)
	// Failed extract must not leave a partial file at the canonical path
	// (and no leftover sibling tempfiles).
	_, statErr := os.Stat(dst)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	leftovers, err := filepath.Glob(filepath.Join(dir, ".extract-*"))
	require.NoError(t, err)
	require.Empty(t, leftovers, "tempfile not cleaned up")
}

func TestDownload_RetriesOn5xx(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")
	payload := []byte("fake node binary contents")
	archive := FakeArchive(t, version, target, payload)
	archName := target.ArchiveName(version)
	shaLine := SHALine(archive, archName)

	var archiveHits, shaHits atomicCounter
	mux := http.NewServeMux()
	mux.HandleFunc("/"+version+"/"+archName, func(w http.ResponseWriter, _ *http.Request) {
		if archiveHits.Inc() < 2 {
			http.Error(w, "boom", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/"+version+"/SHASUMS256.txt", func(w http.ResponseWriter, _ *http.Request) {
		if shaHits.Inc() < 2 {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(shaLine))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	SetBaseURL(t, server.URL)

	prevRetry := defaultRetry
	defaultRetry = config.Retry{Attempts: 4}
	t.Cleanup(func() { defaultRetry = prevRetry })

	hostPath, err := Download(t.Context(), t.TempDir(), version, target)
	require.NoError(t, err)
	require.FileExists(t, hostPath)
	require.GreaterOrEqual(t, int(archiveHits.Load()), 2)
	require.GreaterOrEqual(t, int(shaHits.Load()), 2)
}

type atomicCounter struct{ v atomic.Int32 }

func (c *atomicCounter) Inc() int32  { return c.v.Add(1) }
func (c *atomicCounter) Load() int32 { return c.v.Load() }
