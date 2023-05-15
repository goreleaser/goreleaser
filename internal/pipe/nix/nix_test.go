package nix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRunPipePullRequest(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Nix: []config.Nix{
				{
					Name: "foo",
					IDs:  []string{"foo"},
					Tap: config.RepoRef{
						Owner:  "foo",
						Name:   "bar",
						Branch: "update-{{.Version}}",
						PullRequest: config.PullRequest{
							Enabled: true,
						},
					},
				},
			},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, goarch := range []string{"amd64", "arm64", "386"} {
			path := filepath.Join(folder, "dist/foo_"+goos+goarch+".tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "foo_" + goos + "_" + goarch + ".tar.gz",
				Path:   path,
				Goos:   goos,
				Goarch: goarch,
				// Goamd64: "v1",
				Type: artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})

			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())

		}
	}

	client := client.NewMock()
	pipe := Pipe{
		prefetcher: fakeNixShaPrefetcher{
			"https://dummyhost/download/v1.2.1/foo_linux_amd64.tar.gz":  "sha1",
			"https://dummyhost/download/v1.2.1/foo_linux_arm64.tar.gz":  "sha2",
			"https://dummyhost/download/v1.2.1/foo_darwin_amd64.tar.gz": "sha3",
			"https://dummyhost/download/v1.2.1/foo_darwin_arm64.tar.gz": "sha4",
		},
	}
	require.NoError(t, pipe.runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	require.True(t, client.OpenedPullRequest)
	golden.RequireEqualExt(t, []byte(client.Content), ".nix")
}

type fakeNixShaPrefetcher map[string]string

func (m fakeNixShaPrefetcher) Prefetch(url string) (string, error) {
	return m[url], nil
}
