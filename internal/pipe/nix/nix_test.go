package nix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRunPipe(t *testing.T) {
	for _, tt := range []struct {
		name string
		nix  config.Nix
	}{
		{
			name: "open-pr",
			nix: config.Nix{
				Name:        "foo",
				IDs:         []string{"foo"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				Path:        "pkgs/foo.nix",
				Repository: config.RepoRef{
					Owner:  "foo",
					Name:   "bar",
					Branch: "update-{{.Version}}",
					PullRequest: config.PullRequest{
						Enabled: true,
					},
				},
			},
		},
		{
			name: "wrapped-in-dir",
			nix: config.Nix{
				Name:        "wrapped-in-dir",
				IDs:         []string{"wrapped-in-dir"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				PostInstall: `
					echo "do something"
				`,
				Install: `
					mkdir -p $out/bin
					cp foo $out/bin/foo
				`,
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "unibin",
			nix: config.Nix{
				Name:        "unibin",
				IDs:         []string{"unibin"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "unibin-replaces",
			nix: config.Nix{
				Name:        "unibin-replaces",
				IDs:         []string{"unibin-replaces"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "partial",
			nix: config.Nix{
				Name: "partial",
				IDs:  []string{"partial"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
					Nix:         []config.Nix{tt.nix},
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
			)
			createFakeArtifact := func(id, goos, goarch string, extra map[string]any) {
				path := filepath.Join(folder, "dist/foo_"+goos+goarch+".tar.gz")
				art := artifact.Artifact{
					Name:    "foo_" + goos + "_" + goarch + ".tar.gz",
					Path:    path,
					Goos:    goos,
					Goarch:  goarch,
					Goamd64: "v1",
					Type:    artifact.UploadableArchive,
					Extra: map[string]interface{}{
						artifact.ExtraID:       id,
						artifact.ExtraFormat:   "tar.gz",
						artifact.ExtraBinaries: []string{"foo"},
					},
				}
				for k, v := range extra {
					art.Extra[k] = v
				}
				ctx.Artifacts.Add(&art)

				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			createFakeArtifact("unibin-replaces", "darwin", "all", map[string]any{artifact.ExtraReplaces: true})
			createFakeArtifact("unibin", "darwin", "all", nil)
			for _, goos := range []string{"linux", "darwin", "windows"} {
				for _, goarch := range []string{"amd64", "arm64", "386"} {
					if goos+goarch == "darwin386" {
						continue
					}
					if goarch == "amd64" {
						createFakeArtifact("partial", goos, goarch, nil)
					}
					createFakeArtifact("foo", goos, goarch, nil)
					createFakeArtifact("unibin", goos, goarch, nil)
					createFakeArtifact("unibin-replaces", goos, goarch, nil)
					createFakeArtifact("wrapped-in-dir", goos, goarch, map[string]any{artifact.ExtraWrappedIn: "./foo"})
				}
			}

			client := client.NewMock()
			bpipe := NewBuild()
			ppipe := Pipe{
				fakeNixShaPrefetcher{
					"https://dummyhost/download/v1.2.1/foo_linux_amd64.tar.gz":  "sha1",
					"https://dummyhost/download/v1.2.1/foo_linux_arm64.tar.gz":  "sha2",
					"https://dummyhost/download/v1.2.1/foo_darwin_amd64.tar.gz": "sha3",
					"https://dummyhost/download/v1.2.1/foo_darwin_arm64.tar.gz": "sha4",
					"https://dummyhost/download/v1.2.1/foo_darwin_all.tar.gz":   "sha5",
				},
			}
			require.NoError(t, bpipe.Default(ctx))
			require.NoError(t, bpipe.runAll(ctx, client))
			bts, err := os.ReadFile(ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).Paths()[0])
			require.NoError(t, err)
			golden.RequireEqualExt(t, bts, "_build.nix")
			require.NoError(t, ppipe.publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualExt(t, []byte(client.Content), "_publish.nix")
			require.NotContains(t, client.Content, strings.Repeat("0", 52))
			if tt.nix.Repository.PullRequest.Enabled {
				require.True(t, client.OpenedPullRequest)
			}
			if tt.nix.Path != "" {
				require.Equal(t, tt.nix.Path, client.Path)
			}
		})
	}
}

type fakeNixShaPrefetcher map[string]string

func (m fakeNixShaPrefetcher) Prefetch(url string) (string, error) {
	return m[url], nil
}
