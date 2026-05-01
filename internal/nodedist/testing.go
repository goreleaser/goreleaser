package nodedist

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// SetBaseURL overrides the nodejs.org/dist base URL for the duration
// of t and restores the previous value on cleanup. Intended for tests
// — including tests living in other packages — that need to point the
// downloader at a stub server.
func SetBaseURL(t *testing.T, url string) {
	t.Helper()
	prev := distBaseURL
	distBaseURL = url
	t.Cleanup(func() { distBaseURL = prev })
}

// FakeArchive returns the bytes of a minimal nodejs tarball suitable
// for testing Download against the given version+target. The archive
// contains a single `node-<version>-<target>/bin/node` entry holding
// payload. For windows targets, payload is returned unchanged because
// the published windows distribution is the bare node.exe.
func FakeArchive(t *testing.T, version string, target Target, payload []byte) []byte {
	t.Helper()
	if target.IsWindows() {
		return payload
	}
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

// NewServer stands up an httptest.Server that serves the given
// path-to-body map, registered for cleanup on t.
func NewServer(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for p, body := range files {
		mux.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(body)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// StubRelease registers a synthetic SHA for (version, archiveName) in
// the embedded release map for the duration of t. The reported SHA is
// the SHA-256 of payload, so callers can serve payload from a stub
// http server and have Download accept it. Cleanup restores any
// pre-existing value.
func StubRelease(t *testing.T, version, archiveName string, payload []byte) {
	t.Helper()
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	files, ok := Releases()[version]
	if !ok {
		files = map[string]string{}
		Releases()[version] = files
	}
	prev, hadFile := files[archiveName]
	files[archiveName] = sha
	t.Cleanup(func() {
		if hadFile {
			files[archiveName] = prev
		} else {
			delete(files, archiveName)
		}
	})
}

// writeBuffer is a minimal io.Writer-backed bytes buffer; using an
// inline definition avoids dragging in bytes.Buffer just to keep the
// helper self-contained.
type writeBuffer struct{ b []byte }

func (w *writeBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
