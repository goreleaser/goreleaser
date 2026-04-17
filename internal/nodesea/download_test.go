package nodesea

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTargetGoosGoarch(t *testing.T) {
	cases := map[Target]struct{ os, arch string }{
		"linux-x64":    {"linux", "amd64"},
		"linux-arm64":  {"linux", "arm64"},
		"darwin-x64":   {"darwin", "amd64"},
		"darwin-arm64": {"darwin", "arm64"},
		"win-x64":      {"windows", "amd64"},
		"win-arm64":    {"windows", "arm64"},
	}
	for tgt, want := range cases {
		t.Run(string(tgt), func(t *testing.T) {
			require.Equal(t, want.os, tgt.Goos())
			require.Equal(t, want.arch, tgt.Goarch())
		})
	}
}

func TestArchiveAndHostBinaryName(t *testing.T) {
	require.Equal(t, "node-v22.10.0-linux-x64.tar.gz", Target("linux-x64").archiveName("v22.10.0"))
	require.Equal(t, "win-x64/node.exe", Target("win-x64").archiveName("v22.10.0"))
	require.Equal(t, "node", Target("linux-x64").hostBinaryName())
	require.Equal(t, "node.exe", Target("win-x64").hostBinaryName())
}

// fakeNode returns the bytes of a tar.gz archive containing a single
// `node-<ver>-<target>/bin/node` entry with the given payload.
func fakeNode(t *testing.T, version string, target Target, payload []byte) []byte {
	t.Helper()
	var buf writeBuffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	want := fmt.Sprintf("node-%s-%s/bin/node", version, target)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: want,
		Mode: 0o755,
		Size: int64(len(payload)),
	}))
	_, err := tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.b
}

// writeBuffer is a minimal io.Writer-backed bytes buffer; using an
// inline definition avoids dragging in bytes.Buffer just to keep the
// helper self-contained.
type writeBuffer struct{ b []byte }

func (w *writeBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

func newDistServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, body := range files {
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(body)
		})
	}
	return httptest.NewServer(mux)
}

func TestDownloadHost_Linux(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")
	payload := []byte("fake node binary contents")
	archive := fakeNode(t, version, target, payload)
	sum := sha256.Sum256(archive)
	shaLine := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), target.archiveName(version))

	server := newDistServer(t, map[string][]byte{
		"/" + version + "/" + target.archiveName(version): archive,
		"/" + version + "/SHASUMS256.txt":                 []byte(shaLine),
	})
	defer server.Close()

	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	prev := distBaseURL
	distBaseURL = server.URL
	t.Cleanup(func() { distBaseURL = prev })

	cache := t.TempDir()
	hostPath, err := downloadHost(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.FileExists(t, hostPath)
	got, err := os.ReadFile(hostPath)
	require.NoError(t, err)
	require.Equal(t, payload, got)

	// Second call hits cache, should not re-fetch.
	server.Close()
	hostPath2, err := downloadHost(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.Equal(t, hostPath, hostPath2)
}

func TestDownloadHost_Windows(t *testing.T) {
	const version = "v22.10.0"
	target := Target("win-x64")
	payload := []byte("fake node.exe contents")
	sum := sha256.Sum256(payload)
	shaLine := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), target.archiveName(version))

	server := newDistServer(t, map[string][]byte{
		"/" + version + "/win-x64/node.exe": payload,
		"/" + version + "/SHASUMS256.txt":   []byte(shaLine),
	})
	defer server.Close()

	prev := distBaseURL
	distBaseURL = server.URL
	t.Cleanup(func() { distBaseURL = prev })

	cache := t.TempDir()
	hostPath, err := downloadHost(t.Context(), cache, version, target)
	require.NoError(t, err)
	require.FileExists(t, hostPath)
	require.Equal(t, "node.exe", filepath.Base(hostPath))
	got, err := os.ReadFile(hostPath)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestDownloadHost_BadSHA(t *testing.T) {
	const version = "v22.10.0"
	target := Target("linux-x64")
	archive := fakeNode(t, version, target, []byte("payload"))
	shaLine := "deadbeef  " + target.archiveName(version) + "\n"

	server := newDistServer(t, map[string][]byte{
		"/" + version + "/" + target.archiveName(version): archive,
		"/" + version + "/SHASUMS256.txt":                 []byte(shaLine),
	})
	defer server.Close()

	prev := distBaseURL
	distBaseURL = server.URL
	t.Cleanup(func() { distBaseURL = prev })

	_, err := downloadHost(t.Context(), t.TempDir(), version, target)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA-256 mismatch")
}

func TestCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/somewhere")
	dir, err := CacheDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join("/tmp/somewhere", "goreleaser", "node"), dir)
}
