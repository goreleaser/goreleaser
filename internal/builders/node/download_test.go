package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/jarcoal/httpmock"
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

			httpmock.Activate()
			t.Cleanup(httpmock.DeactivateAndReset)
			httpmock.RegisterResponder(
				http.MethodGet,
				"https://nodejs.org/dist/"+version+"/"+name+".tar.gz",
				httpmock.NewBytesResponder(http.StatusOK, archive.Bytes()),
			)

			dir := t.TempDir()
			path, err := downloadHostBinary(t.Context(), version, target, dir)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, "node"), path)
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, contents, string(got))
			require.Equal(t, 1, httpmock.GetTotalCallCount())
		})
	}
}
