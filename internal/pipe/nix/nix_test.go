package nix

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("no-nix", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		require.True(t, New().Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Nix: []config.Nix{{}},
		}, testctx.Skip(skips.Nix))))
	})
	t.Run("nix-all-good", func(t *testing.T) {
		require.False(t, New().Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Nix: []config.Nix{{}},
		})))
	})
}

func TestHasher(t *testing.T) {
	t.Run("reference vector", func(t *testing.T) {
		// produced by `nix-hash --type sha256 --flat --base32 testdata/file.bin`.
		sha, err := realHasher.Hash("./testdata/file.bin")
		require.NoError(t, err)
		require.Equal(t, "1n7yy95h81rziah4ppi64kr6fphwxjiq8cl70fpfrqvr0ml1xbcl", sha)
	})

	t.Run("all zeroes", func(t *testing.T) {
		require.Equal(t, zeroHash, nixBase32(make([]byte, sha256.Size)))
	})

	// the vectors above only emit 27 of the 32 characters; with this one, a
	// typo anywhere in the alphabet fails the test.
	t.Run("full alphabet", func(t *testing.T) {
		// printf goreleaser > f && nix-hash --type sha256 --flat --base32 f
		sum := sha256.Sum256([]byte("goreleaser"))
		require.Equal(t, "19a48kbsv6pr8x1bnbbs61f80wilnc2dcgahb70xbmb73pr0s7sk", nixBase32(sum[:]))
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := realHasher.Hash("./testdata/does-not-exist.bin")
		require.Error(t, err)
	})
}

func TestRunPipe(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		expectDefaultErrorIs error
		expectRunError       func(testing.TB, error)
		expectPublishErrorIs error
		nix                  config.Nix
	}{
		{
			name: "minimal",
			nix: config.Nix{
				IDs: []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "invalid license",
			expectDefaultErrorIs: errInvalidLicense,
			nix: config.Nix{
				IDs:     []string{"foo"},
				License: "mitt",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "deps",
			nix: config.Nix{
				IDs: []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				Dependencies: []config.NixDependency{
					{Name: "fish"},
					{Name: "bash"},
					linuxDep("ttyd"),
					darwinDep("chromium"),
				},
			},
		},
		{
			name: "extra-install",
			nix: config.Nix{
				IDs: []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
				Dependencies: []config.NixDependency{
					{Name: "fish"},
					{Name: "bash"},
					linuxDep("ttyd"),
					darwinDep("chromium"),
				},
				ExtraInstall: "installManPage ./manpages/foo.1.gz",
			},
		},
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
				ExtraInstall: "installManPage ./manpages/foo.1.gz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "main-program",
			nix: config.Nix{
				Name:        "main-program",
				IDs:         []string{"foo"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				MainProgram: "drumroll-{{ .Version }}",
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
			name:           "bad-main-program-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name:        "foo",
				MainProgram: "{{ .Nope }}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "zip",
			nix: config.Nix{
				Name:        "foozip",
				IDs:         []string{"foo-zip"},
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
			name: "zip-with-dependencies",
			nix: config.Nix{
				Name:        "foozip",
				IDs:         []string{"foo-zip"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				Dependencies: []config.NixDependency{
					{Name: "git"},
				},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "zip-and-tar",
			nix: config.Nix{
				Name:        "foozip",
				IDs:         []string{"zip-and-tar"},
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
			name:           "unibin",
			expectRunError: requireErrorIs(ErrMultipleArchivesSamePlatform),
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
			name: "no-archives",
			expectRunError: func(tb testing.TB, err error) {
				tb.Helper()
				var notFound errNoArchivesFound
				require.ErrorAs(tb, err, &notFound)
				require.Equal(tb, errNoArchivesFound{
					goamd64: "v2",
					ids:     []string{"nopenopenope"},
				}, notFound)
			},
			nix: config.Nix{
				Name:    "no-archives",
				IDs:     []string{"nopenopenope"},
				Goamd64: "v2",
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
			name: "binary",
			nix: config.Nix{
				Name:        "foobin",
				IDs:         []string{"binary"},
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
			name: "binary-with-dependencies",
			nix: config.Nix{
				Name:        "foobin",
				IDs:         []string{"binary"},
				Description: "my test",
				Homepage:    "https://goreleaser.com",
				License:     "mit",
				Dependencies: []config.NixDependency{
					{Name: "curl"},
				},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "archive-and-binary",
			nix: config.Nix{
				Name:        "foo",
				IDs:         []string{"archive-and-binary"},
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
			name: "binary-nested-name",
			nix: config.Nix{
				Name: "foobin",
				IDs:  []string{"binary-nested-name"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "universal-binary-partial-archive",
			expectRunError: requireErrorIs(ErrMixedArchivesAndBinaries),
			nix: config.Nix{
				Name: "foo",
				IDs:  []string{"universal-binary-partial-archive"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "binary-names-differ",
			expectRunError: requireBinaryNamesDiffer,
			nix: config.Nix{
				Name: "foo",
				IDs:  []string{"binary-names-differ"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "binary-names-differ-reversed",
			expectRunError: requireBinaryNamesDiffer,
			nix: config.Nix{
				Name: "foo",
				IDs:  []string{"binary-names-differ-reversed"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "binary-names-differ-custom-install",
			nix: config.Nix{
				Name: "foo",
				IDs:  []string{"binary-names-differ"},
				Install: `
					mkdir -p $out/bin
					install -m755 $src $out/bin/foo
				`,
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "mixed-archive-binary",
			expectRunError: requireErrorIs(ErrMixedArchivesAndBinaries),
			nix: config.Nix{
				Name: "foo",
				IDs:  []string{"mixed-archive-binary"},
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
		{
			name:           "no-repo-name",
			expectRunError: requireErrorIs(errNoRepoName),
			nix: config.Nix{
				Name: "doesnotmatter",
				Repository: config.RepoRef{
					Owner: "foo",
				},
			},
		},
		{
			name:           "bad-name-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name: "{{ .Nope }}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-description-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Description: "{{ .Nope }}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-homepage-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Homepage: "{{ .Nope }}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-repo-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name: "doesnotmatter",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "{{ .Nope }}",
				},
			},
		},
		{
			name:           "bad-skip-upload-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name:       "doesnotmatter",
				SkipUpload: "{{ .Nope }}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-install-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name:    "foo",
				Install: `{{.NoInstall}}`,
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-post-install-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name:        "foo",
				PostInstall: `{{.NoPostInstall}}`,
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-path-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name: "foo",
				Path: `{{.Foo}}/bar/foo.nix`,
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:           "bad-release-url-tmpl",
			expectRunError: testlib.RequireTemplateError,
			nix: config.Nix{
				Name:        "foo",
				URLTemplate: "{{.BadURL}}",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload",
			expectPublishErrorIs: errSkipUpload,
			nix: config.Nix{
				Name:       "doesnotmatter",
				SkipUpload: "true",
				IDs:        []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload-auto",
			expectPublishErrorIs: errSkipUploadAuto,
			nix: config.Nix{
				Name:       "doesnotmatter",
				SkipUpload: "auto",
				IDs:        []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
					Nix:         []config.Nix{tt.nix},
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
				testctx.WithSemver(1, 2, 1, "rc1"))

			createFakeArtifact := func(id, goos, goarch, goamd64, goarm, format string, extra map[string]any) {
				if goarch != "arm" {
					goarm = ""
				}
				if goarch != "amd64" {
					goamd64 = ""
				}
				name := "foo_" + goos + "_" + goarch + goamd64 + goarm + "." + format
				path := filepath.Join(folder, "dist", name)
				art := artifact.Artifact{
					Name:    name,
					Path:    path,
					Goos:    goos,
					Goarch:  goarch,
					Goarm:   goarm,
					Goamd64: goamd64,
					Type:    artifact.UploadableArchive,
					Extra: map[string]any{
						artifact.ExtraID:        id,
						artifact.ExtraFormat:    format,
						artifact.ExtraBinaries:  []string{"foo"},
						artifact.ExtraWrappedIn: "",
					},
				}
				maps.Copy(art.Extra, extra)
				ctx.Artifacts.Add(&art)
			}

			createFakeBinary := func(id, goos, goarch, goamd64, goarm string, extra map[string]any) {
				name := "foo_" + goos + "_" + goarch + goamd64 + goarm
				art := artifact.Artifact{
					Name:    name,
					Path:    filepath.Join(folder, "dist", name),
					Goos:    goos,
					Goarch:  goarch,
					Goarm:   goarm,
					Goamd64: goamd64,
					Type:    artifact.UploadableBinary,
					Extra: map[string]any{
						artifact.ExtraID:     id,
						artifact.ExtraFormat: "binary",
						artifact.ExtraBinary: "foo",
					},
				}
				maps.Copy(art.Extra, extra)
				ctx.Artifacts.Add(&art)
			}

			createFakeBinary("binary", "linux", "amd64", "v1", "", nil)
			createFakeBinary("binary", "linux", "arm64", "", "", nil)
			createFakeBinary("binary", "linux", "arm", "", "7", nil)
			createFakeBinary("binary", "darwin", "all", "", "", map[string]any{artifact.ExtraReplaces: true})
			createFakeBinary("binary", "darwin", "amd64", "v1", "", map[string]any{artifact.ExtraReplaces: false})
			createFakeBinary("binary", "darwin", "arm64", "", "", map[string]any{artifact.ExtraReplaces: false})
			createFakeBinary("binary", "windows", "amd64", "v1", "", nil)
			for _, goos := range []string{"linux", "darwin"} {
				for _, goarch := range []string{"amd64", "arm64"} {
					createFakeArtifact("archive-and-binary", goos, goarch, "v1", "", "tar.gz", nil)
					createFakeBinary("archive-and-binary", goos, goarch, map[string]string{"amd64": "v1"}[goarch], "", nil)
				}
			}
			createFakeBinary("binary-nested-name", "linux", "arm64", "", "", map[string]any{artifact.ExtraBinary: "bin/foo"})
			createFakeBinary("universal-binary-partial-archive", "darwin", "all", "", "", map[string]any{artifact.ExtraReplaces: true})
			createFakeArtifact("universal-binary-partial-archive", "darwin", "amd64", "v1", "", "tar.gz", nil)
			createFakeArtifact("mixed-archive-binary", "darwin", "arm64", "", "", "zip", nil)
			createFakeBinary("mixed-archive-binary", "linux", "arm64", "", "", nil)
			// same platforms, differing binary names, in both artifact orders.
			createFakeBinary("binary-names-differ", "linux", "arm64", "", "", map[string]any{artifact.ExtraBinary: "foo-linux"})
			createFakeBinary("binary-names-differ", "darwin", "arm64", "", "", map[string]any{artifact.ExtraBinary: "foo-darwin"})
			createFakeBinary("binary-names-differ-reversed", "darwin", "arm64", "", "", map[string]any{artifact.ExtraBinary: "foo-darwin"})
			createFakeBinary("binary-names-differ-reversed", "linux", "arm64", "", "", map[string]any{artifact.ExtraBinary: "foo-linux"})

			createFakeArtifact("unibin-replaces", "darwin", "all", "", "", "tar.gz", map[string]any{artifact.ExtraReplaces: true})
			createFakeArtifact("unibin", "darwin", "all", "", "", "tgz", nil)
			for _, goos := range []string{"linux", "darwin", "windows"} {
				for _, goarch := range []string{"amd64", "arm64", "386", "arm"} {
					if goos+goarch == "darwin386" {
						continue
					}
					if goarch == "amd64" {
						createFakeArtifact("partial", goos, goarch, "v1", "", "tar.xz", nil)
						createFakeArtifact("foo", goos, goarch, "v1", "", "txz", nil)
						createFakeArtifact("unibin", goos, goarch, "v1", "", "tar.zst", nil)
						if goos != "darwin" {
							createFakeArtifact("unibin-replaces", goos, goarch, "v1", "", "tzst", nil)
						}
						createFakeArtifact("wrapped-in-dir", goos, goarch, "v1", "", "tar", map[string]any{artifact.ExtraWrappedIn: "./foo_" + goarch})
						createFakeArtifact("foo-zip", goos, goarch, "v1", "", "zip", nil)
						continue
					}
					if goarch == "arm" {
						if goos != "linux" {
							continue
						}
						createFakeArtifact("foo", goos, goarch, "", "6", "tar.gz", nil)
						createFakeArtifact("foo", goos, goarch, "", "7", "tar.gz", nil)
						createFakeArtifact("foo-zip", goos, goarch, "", "", "zip", nil)
						createFakeArtifact("unibin-replaces", goos, goarch, "", "", "tar.gz", nil)
						continue
					}
					createFakeArtifact("foo", goos, goarch, "", "", "tar.gz", nil)
					createFakeArtifact("unibin", goos, goarch, "", "", "tar.gz", nil)
					if goos != "darwin" {
						createFakeArtifact("unibin-replaces", goos, goarch, "", "", "tar.gz", nil)
					}
					createFakeArtifact("wrapped-in-dir", goos, goarch, "", "", "tar.gz", map[string]any{artifact.ExtraWrappedIn: "./foo_" + goarch})
					createFakeArtifact("foo-zip", goos, goarch, "v1", "", "zip", nil)
					if goos == "darwin" {
						createFakeArtifact("zip-and-tar", goos, goarch, "v1", "", "zip", nil)
					} else {
						createFakeArtifact("zip-and-tar", goos, goarch, "v1", "", "tar.gz", nil)
					}
				}
			}

			readSummary := testlib.CaptureSummary(t)
			client := client.NewMock()
			bpipe := Pipe{
				alwaysZeroHasher{},
			}
			ppipe := Pipe{
				fakeHasher{
					"foo_linux_amd64v1.txz":     "sha1",
					"foo_linux_amd64v1.tzst":    "sha1",
					"foo_linux_amd64v1.tar.xz":  "sha1",
					"foo_linux_amd64v1.tar":     "sha1",
					"foo_linux_amd64v1.tar.gz":  "sha1",
					"foo_linux_arm64.tar.gz":    "sha2",
					"foo_darwin_amd64v1.txz":    "sha3",
					"foo_darwin_amd64v1.tar":    "sha3",
					"foo_darwin_amd64v1.tar.xz": "sha3",
					"foo_darwin_arm64.tar.gz":   "sha4",
					"foo_darwin_all.tgz":        "sha5",
					"foo_darwin_all.tar.gz":     "sha5",
					"foo_linux_arm6.tar.gz":     "sha6",
					"foo_linux_arm7.tar.gz":     "sha7",
					"foo_linux_amd64v1.zip":     "sha8",
					"foo_linux_arm64.zip":       "sha9",
					"foo_darwin_amd64v1.zip":    "sha10",
					"foo_darwin_arm64.zip":      "sha11",
					"foo_darwin_all.zip":        "sha12",
					"foo_linux_arm6.zip":        "sha13",
					"foo_linux_arm7.zip":        "sha14",
					"foo_linux_386.zip":         "sha15",
					"foo_linux_386.tar.gz":      "sha16",
					"foo_darwin_amd64v1.tar.gz": "sha21",
					"foo_linux_arm.tar.gz":      "sha22",
					"foo_linux_arm.zip":         "sha23",
					"foo_darwin_arm64":          "sha24",
					"foo_linux_amd64v1":         "sha17",
					"foo_linux_arm64":           "sha18",
					"foo_linux_arm7":            "sha19",
					"foo_darwin_all":            "sha20",
				},
			}

			// default
			if tt.expectDefaultErrorIs != nil {
				err := bpipe.Default(ctx)
				require.ErrorIs(t, err, tt.expectDefaultErrorIs)
				return
			}
			require.NoError(t, bpipe.Default(ctx))

			// run
			if tt.expectRunError != nil {
				tt.expectRunError(t, bpipe.runAll(ctx, client))
				return
			}
			require.NoError(t, bpipe.runAll(ctx, client))
			bts, err := os.ReadFile(ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).Paths()[0])
			require.NoError(t, err)
			golden.RequireEqualExt(t, bts, "_build.nix")

			// publish
			if tt.expectPublishErrorIs != nil {
				err := ppipe.publishAll(ctx, client)
				require.ErrorIs(t, err, tt.expectPublishErrorIs)
				return
			}
			require.NoError(t, ppipe.publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualExt(t, []byte(client.Content), "_publish.nix")
			require.NotContains(t, client.Content, strings.Repeat("0", 52))

			if tt.nix.Repository.PullRequest.Enabled {
				require.True(t, client.OpenedPullRequest)
				require.True(t, client.SyncedFork)
				require.Contains(t, strings.Join(readSummary(), "\n"), "): https://github.com/goreleaser/goreleaser/pull/1")
			}
			if tt.nix.Path != "" {
				require.Equal(t, tt.nix.Path, client.Path)
			} else {
				if tt.nix.Name == "" {
					tt.nix.Name = "foo"
				}
				require.Equal(t, "pkgs/"+tt.nix.Name+"/default.nix", client.Path)
			}
		})
	}
}

func TestRunPipeSomeSkipped(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Nix: []config.Nix{
				{
					// no repository.name: gets skipped.
					Name: "skipped",
				},
				{
					Name: "bar",
					IDs:  []string{"foo"},
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
	)

	for _, goarch := range []string{"amd64", "arm64"} {
		name := "foo_linux_" + goarch + ".tar.gz"
		path := filepath.Join(folder, "dist", name)
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   name,
			Path:   path,
			Goos:   "linux",
			Goarch: goarch,
			Type:   artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:        "foo",
				artifact.ExtraFormat:    "tar.gz",
				artifact.ExtraBinaries:  []string{"foo"},
				artifact.ExtraWrappedIn: "",
			},
		})
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	p := Pipe{alwaysZeroHasher{}}
	require.NoError(t, p.Default(ctx))
	testlib.AssertSkipped(t, p.runAll(ctx, client.NewMock()))
	require.Len(
		t,
		ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).List(),
		1,
		"second nix package should have run",
	)
}

func TestPublishFormatsContent(t *testing.T) {
	testlib.SkipIfWindows(t, "nix.format won't work on Windows")

	for _, formatter := range []string{"alejandra", "nixfmt"} {
		t.Run(formatter, func(t *testing.T) {
			bin := t.TempDir()
			require.NoError(t, os.WriteFile(
				filepath.Join(bin, formatter),
				[]byte("#!/bin/sh\nprintf '\\n# formatted by "+formatter+"\\n' >> \"$1\"\n"),
				0o755,
			))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(
				t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
			)

			archiveName := "foo_linux_amd64v1.txz"
			archivePath := filepath.Join(folder, "dist", archiveName)
			require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
			require.NoError(t, os.WriteFile(archivePath, nil, 0o644))
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    archiveName,
				Path:    archivePath,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:        "foo",
					artifact.ExtraFormat:    "txz",
					artifact.ExtraBinaries:  []string{"foo"},
					artifact.ExtraWrappedIn: "",
				},
			})

			nix := config.Nix{
				Name:                  "foo",
				Path:                  "pkgs/foo/default.nix",
				IDs:                   []string{"foo"},
				Goamd64:               "v1",
				Formatter:             formatter,
				CommitAuthor:          config.CommitAuthor{Name: "goreleaserbot", Email: "bot@goreleaser.com"},
				CommitMessageTemplate: "publish",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			}
			pkg := &artifact.Artifact{
				Name: "default.nix",
				Type: artifact.Nixpkg,
				Extra: map[string]any{
					nixConfigExtra: nix,
				},
			}
			client := client.NewMock()

			require.NoError(t, doPublish(ctx, fakeHasher{archiveName: "publication-sha"}, client, pkg))
			require.Contains(t, client.Content, `x86_64-linux = "publication-sha";`)
			require.Contains(t, client.Content, "# formatted by "+formatter)
		})
	}
}

func TestPreparePkgEscapesDescription(t *testing.T) {
	for name, tt := range map[string]struct {
		description string
		expect      string
	}{
		"quotes and interpolation": {
			description: `Say "hello" from C:\tools and ${system}`,
			expect:      `description = "Say \"hello\" from C:\\tools and \${system}";`,
		},
		// Control characters need escaping because doBuildPkg sanitizes the
		// rendered derivation line by line, trimming trailing spaces and
		// dropping carriage returns, which would otherwise silently mangle a
		// multi-line description.
		"control characters": {
			description: "line1\nline2\twith  \r\ntrailing",
			expect:      `description = "line1\nline2\twith  \r\ntrailing";`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(
				t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
			)

			archiveName := "foo_linux_amd64v1.txz"
			archivePath := filepath.Join(folder, "dist", archiveName)
			require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
			require.NoError(t, os.WriteFile(archivePath, nil, 0o644))
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    archiveName,
				Path:    archivePath,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:        "foo",
					artifact.ExtraFormat:    "txz",
					artifact.ExtraBinaries:  []string{"foo"},
					artifact.ExtraWrappedIn: "",
				},
			})

			content, err := preparePkg(ctx, config.Nix{
				Name:        "foo",
				IDs:         []string{"foo"},
				Goamd64:     "v1",
				Description: tt.description,
			}, client.NewMock(), fakeHasher{archiveName: "sha"})
			require.NoError(t, err)
			require.Contains(t, content, tt.expect)
		})
	}
}

// The whole rendered derivation goes through the template engine once more,
// so a description that renders to text containing `{{` must survive that
// pass as a literal instead of being evaluated again.
func TestPreparePkgDescriptionIsTemplatedOnce(t *testing.T) {
	for name, tt := range map[string]struct {
		description string
		expect      string
	}{
		"from env": {
			description: `{{ .Env.DESCRIPTION }}`,
			expect:      `description = "Render {{example}} templates";`,
		},
		"needs escaping too": {
			description: `Say "hi" from {{ "{{example}}" }}`,
			expect:      `description = "Say \"hi\" from {{example}}";`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(
				t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: "foo",
				},
				testctx.WithEnv(map[string]string{"DESCRIPTION": "Render {{example}} templates"}),
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
			)

			archiveName := "foo_linux_amd64v1.txz"
			archivePath := filepath.Join(folder, "dist", archiveName)
			require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
			require.NoError(t, os.WriteFile(archivePath, nil, 0o644))
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    archiveName,
				Path:    archivePath,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:        "foo",
					artifact.ExtraFormat:    "txz",
					artifact.ExtraBinaries:  []string{"foo"},
					artifact.ExtraWrappedIn: "",
				},
			})

			nix := config.Nix{
				Name:        "foo",
				IDs:         []string{"foo"},
				Goamd64:     "v1",
				Description: tt.description,
			}
			require.NoError(t, tmpl.New(ctx).ApplyAll(&nix.Description))

			content, err := preparePkg(ctx, nix, client.NewMock(), fakeHasher{archiveName: "sha"})
			require.NoError(t, err)
			require.Contains(t, content, tt.expect)
		})
	}
}

func TestPreparePkgInstallPhaseRunsHooks(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)

	archiveName := "foo_linux_amd64v1.txz"
	archivePath := filepath.Join(folder, "dist", archiveName)
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, nil, 0o644))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    archiveName,
		Path:    archivePath,
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:        "foo",
			artifact.ExtraFormat:    "txz",
			artifact.ExtraBinaries:  []string{"foo"},
			artifact.ExtraWrappedIn: "",
		},
	})

	content, err := preparePkg(ctx, config.Nix{
		Name:        "foo",
		IDs:         []string{"foo"},
		Goamd64:     "v1",
		PostInstall: `printf 'post-install ran\n' > "$out/post-install.marker"`,
	}, client.NewMock(), fakeHasher{archiveName: "sha"})
	require.NoError(t, err)
	require.Contains(t, content, `  installPhase = ''
    runHook preInstall
    mkdir -p $out/bin
    cp -vr ./foo $out/bin/foo
    runHook postInstall
  '';`)
	require.Contains(t, content, `  postInstall = ''
    printf 'post-install ran\n' > "$out/post-install.marker"
  '';`)
}

func TestErrNoArchivesFound(t *testing.T) {
	require.EqualError(t, errNoArchivesFound{
		goamd64: "v1",
		ids:     []string{"foo", "bar"},
	}, "no archives found matching goos=[darwin linux] goarch=[amd64 arm arm64 386] goarm=[6 7] goamd64=v1 ids=[foo bar]")
}

func TestBinInstallFormats(t *testing.T) {
	t.Run("no-deps", func(t *testing.T) {
		golden.RequireEqual(t, []byte(strings.Join(
			binInstallFormats(config.Nix{}, archiveInstallFormat),
			"\n",
		)))
	})
	t.Run("deps", func(t *testing.T) {
		golden.RequireEqual(t, []byte(strings.Join(
			binInstallFormats(config.Nix{
				Dependencies: []config.NixDependency{
					{Name: "fish"},
					{Name: "bash"},
					{Name: "zsh"},
				},
			}, archiveInstallFormat),
			"\n",
		)))
	})
	t.Run("linux-only-deps", func(t *testing.T) {
		golden.RequireEqual(t, []byte(strings.Join(
			binInstallFormats(config.Nix{
				Dependencies: []config.NixDependency{
					linuxDep("foo"),
					linuxDep("bar"),
				},
			}, archiveInstallFormat),
			"\n",
		)))
	})
	t.Run("darwin-only-deps", func(t *testing.T) {
		golden.RequireEqual(t, []byte(strings.Join(
			binInstallFormats(config.Nix{
				Dependencies: []config.NixDependency{
					darwinDep("foo"),
					darwinDep("bar"),
				},
			}, archiveInstallFormat),
			"\n",
		)))
	})
	t.Run("mixed-deps", func(t *testing.T) {
		golden.RequireEqual(t, []byte(strings.Join(
			binInstallFormats(config.Nix{
				Dependencies: []config.NixDependency{
					{Name: "fish"},
					linuxDep("foo"),
					darwinDep("bar"),
				},
			}, archiveInstallFormat),
			"\n",
		)))
	})
}

// requireErrorIs returns a check asserting that the error wraps target.
func requireErrorIs(target error) func(testing.TB, error) {
	return func(tb testing.TB, err error) {
		tb.Helper()
		require.ErrorIs(tb, err, target)
	}
}

// requireBinaryNamesDiffer checks that the error lists the differing binary
// names sorted, so the message does not depend on the artifact order.
func requireBinaryNamesDiffer(tb testing.TB, err error) {
	tb.Helper()
	require.ErrorIs(tb, err, ErrBinaryNamesDiffer)
	require.ErrorContains(tb, err, ErrBinaryNamesDiffer.Error()+": foo-darwin, foo-linux")
}

func darwinDep(s string) config.NixDependency {
	return config.NixDependency{
		Name: s,
		OS:   "darwin",
	}
}

func linuxDep(s string) config.NixDependency {
	return config.NixDependency{
		Name: s,
		OS:   "linux",
	}
}

type fakeHasher map[string]string

// Hash fails on files it has no hash for, so that a fixture missing an entry
// cannot silently drop a platform from the generated shaMap.
func (m fakeHasher) Hash(path string) (string, error) {
	sha, ok := m[filepath.Base(path)]
	if !ok {
		return "", fmt.Errorf("fakeHasher: no hash for %s", filepath.Base(path))
	}
	return sha, nil
}

const zeroHash = "0000000000000000000000000000000000000000000000000000"

type alwaysZeroHasher struct{}

func (alwaysZeroHasher) Hash(string) (string, error) { return zeroHash, nil }

func TestDynamicallyLinked(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Nix: []config.Nix{{
				IDs: []string{"dynlink"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			}},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
	)

	createDynlinkArtifact := func(goos, goarch string, dynLinked bool) {
		name := "foo_" + goos + "_" + goarch + ".tar.gz"
		path := filepath.Join(folder, "dist", name)
		art := artifact.Artifact{
			Name:   name,
			Path:   path,
			Goos:   goos,
			Goarch: goarch,
			Type:   artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:        "dynlink",
				artifact.ExtraFormat:    "tar.gz",
				artifact.ExtraBinaries:  []string{"foo"},
				artifact.ExtraWrappedIn: "",
				artifact.ExtranDynLink:  dynLinked,
			},
		}
		ctx.Artifacts.Add(&art)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	// Create linux archives with dynamic linking
	createDynlinkArtifact("linux", "amd64", true)
	createDynlinkArtifact("linux", "arm64", true)
	// Darwin archives won't be dynamically linked (macOS uses different mechanism)
	createDynlinkArtifact("darwin", "amd64", false)
	createDynlinkArtifact("darwin", "arm64", false)

	client := client.NewMock()
	pipe := Pipe{alwaysZeroHasher{}}

	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.runAll(ctx, client))

	nixpkgs := ctx.Artifacts.Filter(artifact.ByType(artifact.Nixpkg)).List()
	require.Len(t, nixpkgs, 1)

	content, err := os.ReadFile(nixpkgs[0].Path)
	require.NoError(t, err)

	golden.RequireEqual(t, content)
}

func TestFormat(t *testing.T) {
	testlib.SkipIfWindows(t, "nix.format won't work on Windows")

	t.Run("invalid formatter", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.False(t, format(ctx, "invalid-formatter", "nope.nix"))
	})

	const input = `{  foo = "bar";
							baz = "qux";	}`

	t.Run("alejandra", func(t *testing.T) {
		testlib.CheckPath(t, "alejandra")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(input), 0o644))

		require.True(t, format(ctx, "alejandra", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})

	t.Run("nixfmt", func(t *testing.T) {
		testlib.CheckPath(t, "nixfmt")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(input), 0o644))

		require.True(t, format(ctx, "nixfmt", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})

	t.Run("invalid file", func(t *testing.T) {
		testlib.CheckPath(t, "nixfmt")

		ctx := testctx.Wrap(t.Context())
		path := filepath.Join(t.TempDir(), "test.nix")
		require.NoError(t, os.WriteFile(path, []byte(`{ invalid file`), 0o644))

		require.False(t, format(ctx, "nixfmt", path))

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		golden.RequireEqualExt(t, content, ".nix")
	})
}
