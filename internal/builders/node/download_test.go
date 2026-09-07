package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/stretchr/testify/require"
)

func TestDownloadHostBinary(t *testing.T) {
	const version = "v25.5.0"
	const contents = "node binary fixture"
	for _, target := range []string{"darwin-arm64", "linux-x64"} {
		t.Run(target, func(t *testing.T) {
			name := "node-" + version + "-" + target
			var archive bytes.Buffer
			gz := gzip.NewWriter(&archive)
			tw := tar.NewWriter(gz)
			for _, file := range []struct {
				name string
				data string
			}{
				{name + "/README.md", "not the executable"},
				{name + "/bin/node", contents},
			} {
				require.NoError(t, tw.WriteHeader(&tar.Header{
					Name: file.name,
					Mode: 0o755,
					Size: int64(len(file.data)),
				}))
				_, err := tw.Write([]byte(file.data))
				require.NoError(t, err)
			}
			require.NoError(t, tw.Close())
			require.NoError(t, gz.Close())

			previous := nodedist.Releases
			nodedist.Releases = func() map[string]map[string]string {
				return map[string]map[string]string{
					version: {
						name + ".tar.gz": fmt.Sprintf("%x", sha256.Sum256(archive.Bytes())),
					},
				}
			}
			t.Cleanup(func() { nodedist.Releases = previous })

			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.Host != "nodejs.org" || r.URL.Path != "/"+version+"/"+name+".tar.gz" {
					t.Errorf("unexpected request: %s %s%s", r.Method, r.Host, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				if _, err := w.Write(archive.Bytes()); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			previousTransport := http.DefaultClient.Transport
			http.DefaultClient.Transport = nodeDistTestTransport{host: server.Listener.Addr().String()}
			t.Cleanup(func() { http.DefaultClient.Transport = previousTransport })

			dir := t.TempDir()
			path, err := downloadHostBinary(t.Context(), version, target, dir)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, "node"), path)
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, contents, string(got))
			require.Equal(t, int32(1), requests.Load())
		})
	}
}

type nodeDistTestTransport struct {
	host string
}

func (t nodeDistTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.host
	return http.DefaultTransport.RoundTrip(req)
}
