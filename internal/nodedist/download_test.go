package nodedist

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	const version = "v22.10.0"
	const archName = "node-v22.10.0-linux-x64.tar.gz"
	payload := []byte("fake archive bytes")
	StubRelease(t, version, archName, payload)

	server := NewServer(t, map[string][]byte{
		"/" + version + "/" + archName: payload,
	})
	SetBaseURL(t, server.URL)

	got, err := Download(t.Context(), version, archName)
	require.NoError(t, err)
	require.FileExists(t, got)
	bts, err := os.ReadFile(got)
	require.NoError(t, err)
	require.Equal(t, payload, bts)
}

func TestDownload_BadSHA(t *testing.T) {
	const version = "v22.10.0"
	const archName = "node-v22.10.0-linux-x64.tar.gz"
	StubRelease(t, version, archName, []byte("expected"))

	server := NewServer(t, map[string][]byte{
		"/" + version + "/" + archName: []byte("actual"),
	})
	SetBaseURL(t, server.URL)

	_, err := Download(t.Context(), version, archName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA-256 mismatch")
}

func TestDownload_UnknownVersion(t *testing.T) {
	_, err := Download(t.Context(), "v0.0.999", "whatever.tar.gz")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no embedded entry")
}

func TestDownload_RetriesOn5xx(t *testing.T) {
	const version = "v22.10.0"
	const archName = "node-v22.10.0-linux-x64.tar.gz"
	payload := []byte("fake archive bytes")
	StubRelease(t, version, archName, payload)

	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/"+version+"/"+archName, func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) < 2 {
			http.Error(w, "boom", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(payload)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	SetBaseURL(t, server.URL)

	prevRetry := defaultRetry
	defaultRetry = config.Retry{Attempts: 4}
	t.Cleanup(func() { defaultRetry = prevRetry })

	got, err := Download(t.Context(), version, archName)
	require.NoError(t, err)
	require.FileExists(t, got)
	require.GreaterOrEqual(t, int(hits.Load()), 2)
}
