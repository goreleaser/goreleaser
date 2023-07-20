package winget

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("should", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})
	t.Run("should not", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Winget: []config.Winget{{}},
		})))
	})
}

func TestRunPipe(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		expectRunErrorIs     error
		expectPublishErrorIs error
		expectPath           string
		winget               config.Winget
	}{
		{
			name:       "minimal",
			expectPath: "manifests/f/Foo/min/1.2.1/Foo.min.",
			winget: config.Winget{
				Name:             "min",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:       "full",
			expectPath: "manifests/b/Beckersoft LTDA/foo/1.2.1",
			winget: config.Winget{
				Name:                "foo",
				Publisher:           "Beckersoft",
				PublisherURL:        "https://carlosbecker.com",
				PublisherSupportURL: "https://carlosbecker.com/support",
				Copyright:           "bla bla bla",
				CopyrightURL:        "https://goreleaser.com/copyright",
				Author:              "Carlos Becker",
				Path:                "manifests/b/Beckersoft LTDA/foo/{{.Version}}",
				Repository:          config.RepoRef{Owner: "foo", Name: "bar"},
				CommitAuthor:        config.CommitAuthor{},
				IDs:                 []string{"foo"},
				Goamd64:             "v1",
				SkipUpload:          "false",
				ShortDescription:    "foo",
				Description: `long foo bar

				yadaa yada yada loooaaasssss

				sss`,
				Homepage:        "https://goreleaser.com",
				License:         "MIT",
				LicenseURL:      "https://goreleaser.com/eula/",
				ReleaseNotesURL: "https://github.com/goreleaser/goreleaser/tags/{{.Tag}}",
				ReleaseNotes:    "{{.Changelog}}",
				Tags:            []string{"foo", "bar"},
			},
		},
		{
			name: "open-pr",
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				IDs:              []string{"foo"},
				Description:      "my test",
				Homepage:         "https://goreleaser.com",
				License:          "mit",
				Path:             "pkgs/foo.winget",
				ShortDescription: "foo bar zaz",
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
			winget: config.Winget{
				Name:             "wrapped-in-dir",
				Publisher:        "Beckersoft",
				IDs:              []string{"wrapped-in-dir"},
				Description:      "my test",
				Homepage:         "https://goreleaser.com",
				License:          "mit",
				LicenseURL:       "https://goreleaser.com/license",
				ReleaseNotesURL:  "https://github.com/goreleaser/goreleaser/tags/{{.Tag}}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "no-archives",
			expectRunErrorIs: errNoArchivesFound{
				goamd64: "v2",
				ids:     []string{"nopenopenope"},
			},
			winget: config.Winget{
				Name:             "no-archives",
				Publisher:        "Beckersoft",
				IDs:              []string{"nopenopenope"},
				Goamd64:          "v2",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "too-many-archives",
			expectRunErrorIs: errMultipleArchives,
			winget: config.Winget{
				Name:             "min",
				Publisher:        "Foo",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name: "partial",
			winget: config.Winget{
				Name:             "partial",
				Publisher:        "Beckersoft",
				IDs:              []string{"partial"},
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "no-repo-name",
			expectRunErrorIs: errNoRepoName,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
				},
			},
		},
		{
			name:             "no-license",
			expectRunErrorIs: errNoLicense,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				ShortDescription: "aa",
				Repository: config.RepoRef{
					Name:  "foo",
					Owner: "foo",
				},
			},
		},
		{
			name:             "no-short-description",
			expectRunErrorIs: errNoShortDescription,
			winget: config.Winget{
				Name:      "doesnotmatter",
				Publisher: "Beckersoft",
				License:   "MIT",
				Repository: config.RepoRef{
					Name:  "foo",
					Owner: "foo",
				},
			},
		},
		{
			name:             "invalid-package-identifier",
			expectRunErrorIs: errInvalidPackageIdentifier,
			winget: config.Winget{
				Name:              "min",
				PackageIdentifier: "foobar",
				Publisher:         "Foo",
				License:           "MIT",
				ShortDescription:  "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "no-publisher",
			expectRunErrorIs: errNoPublisher,
			winget: config.Winget{
				Name:             "min",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-name-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-releasenotes-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				ReleaseNotes:     "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				PublisherURL:     "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-author-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Author:           "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-homepage-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Homepage:         "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-description-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				Description:      "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-short-description-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foobar",
				Publisher:        "Beckersoft",
				ShortDescription: "{{ .Nope }}",
				License:          "MIT",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-repo-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "{{ .Nope }}",
				},
			},
		},
		{
			name:             "bad-skip-upload-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				SkipUpload:       "{{ .Nope }}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-release-notes-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				ReleaseNotesURL:  `https://goo/bar/asdfsd/{{.nope}}`,
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-release-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				URLTemplate:      "{{.BadURL}}",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-path-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				Path:             "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "bad-commit-msg-tmpl",
			expectPublishErrorIs: &template.Error{},
			winget: config.Winget{
				Name:                  "foo",
				Publisher:             "Beckersoft",
				License:               "MIT",
				ShortDescription:      "foo bar zaz",
				CommitMessageTemplate: "{{.Foo}}",
				IDs:                   []string{"foo"},
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-publisher-support-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:                "foo",
				Publisher:           "Beckersoft",
				PublisherSupportURL: "{{.Nope}}",
				License:             "MIT",
				ShortDescription:    "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-copyright-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				Copyright:        "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-copyright-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "{{ .Nope }}",
				Publisher:        "Beckersoft",
				License:          "MIT",
				CopyrightURL:     "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-license-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:             "bad-license-url-tmpl",
			expectRunErrorIs: &template.Error{},
			winget: config.Winget{
				Name:             "foo",
				Publisher:        "Beckersoft",
				License:          "MIT",
				LicenseURL:       "{{ .Nope }}",
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload",
			expectPublishErrorIs: errSkipUpload,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				SkipUpload:       "true",
				License:          "MIT",
				IDs:              []string{"foo"},
				ShortDescription: "foo bar zaz",
				Repository: config.RepoRef{
					Owner: "foo",
					Name:  "bar",
				},
			},
		},
		{
			name:                 "skip-upload-auto",
			expectPublishErrorIs: errSkipUploadAuto,
			winget: config.Winget{
				Name:             "doesnotmatter",
				Publisher:        "Beckersoft",
				License:          "MIT",
				ShortDescription: "foo bar zaz",
				IDs:              []string{"foo"},
				SkipUpload:       "auto",
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
					Winget:      []config.Winget{tt.winget},
				},
				testctx.WithVersion("1.2.1"),
				testctx.WithCurrentTag("v1.2.1"),
				testctx.WithSemver(1, 2, 1, "rc1"),
				testctx.WithDate(time.Date(2023, 6, 12, 20, 32, 10, 12, time.Local)),
			)
			ctx.ReleaseNotes = "the changelog for this release..."
			createFakeArtifact := func(id, goos, goarch, goamd64, goarm string, extra map[string]any) {
				path := filepath.Join(folder, "dist/foo_"+goos+goarch+goamd64+goarm+".zip")
				art := artifact.Artifact{
					Name:    "foo_" + goos + "_" + goarch + goamd64 + goarm + ".zip",
					Path:    path,
					Goos:    goos,
					Goarch:  goarch,
					Goarm:   goarm,
					Goamd64: goamd64,
					Type:    artifact.UploadableArchive,
					Extra: map[string]interface{}{
						artifact.ExtraID:        id,
						artifact.ExtraFormat:    "zip",
						artifact.ExtraBinaries:  []string{"foo.exe"},
						artifact.ExtraWrappedIn: "",
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

			goos := "windows"
			goarch := "amd64"
			createFakeArtifact("partial", goos, goarch, "v1", "", nil)
			createFakeArtifact("foo", goos, goarch, "v1", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "v1", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			goarch = "386"
			createFakeArtifact("foo", goos, goarch, "", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			goarch = "arm64"
			createFakeArtifact("foo", goos, goarch, "", "", nil)
			createFakeArtifact("wrapped-in-dir", goos, goarch, "", "", map[string]any{
				artifact.ExtraWrappedIn: "foo",
				artifact.ExtraBinaries:  []string{"bin/foo.exe"},
			})

			client := client.NewMock()
			pipe := Pipe{}

			// default
			require.NoError(t, pipe.Default(ctx))

			// run
			if tt.expectRunErrorIs != nil {
				err := pipe.runAll(ctx, client)
				require.Error(t, err)
				require.ErrorAs(t, err, &tt.expectPublishErrorIs)
				return
			}

			require.NoError(t, pipe.runAll(ctx, client))
			for _, winget := range ctx.Artifacts.Filter(artifact.Or(
				artifact.ByType(artifact.WingetInstaller),
				artifact.ByType(artifact.WingetVersion),
				artifact.ByType(artifact.WingetDefaultLocale),
			)).List() {
				bts, err := os.ReadFile(winget.Path)
				require.NoError(t, err)
				golden.RequireEqualExtSubfolder(t, bts, extFor(winget.Type))
			}

			// publish
			if tt.expectPublishErrorIs != nil {
				err := pipe.publishAll(ctx, client)
				require.Error(t, err)
				require.ErrorAs(t, err, &tt.expectPublishErrorIs)
				return
			}
			require.NoError(t, pipe.publishAll(ctx, client))
			require.True(t, client.CreatedFile)

			require.Len(t, client.Messages, 3)
			expected := map[string]bool{
				"locale":    false,
				"version":   false,
				"installer": false,
			}
			for _, msg := range client.Messages {
				require.Regexp(t, "New version: \\w+\\.[\\w-]+ 1.2.1", msg)
				for k, v := range expected {
					if strings.Contains(msg, ": add "+k) {
						require.False(t, v, "multiple commits for "+k)
						expected[k] = true
					}
				}
			}

			require.NotEmpty(t, client.Path)
			if tt.expectPath != "" {
				require.Truef(t, strings.HasPrefix(client.Path, tt.expectPath), "expected %q to begin with %q", client.Path, tt.expectPath)
			}

			if tt.winget.Repository.PullRequest.Enabled {
				require.True(t, client.OpenedPullRequest)
			}
		})
	}
}

func TestErrNoArchivesFound(t *testing.T) {
	require.EqualError(t, errNoArchivesFound{
		goamd64: "v1",
		ids:     []string{"foo", "bar"},
	}, "no zip archives found matching goos=[windows] goarch=[amd64 386] goamd64=v1 ids=[foo bar]")
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		Winget:      []config.Winget{{}},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	winget := ctx.Config.Winget[0]
	require.Equal(t, "v1", winget.Goamd64)
	require.NotEmpty(t, winget.CommitMessageTemplate)
	require.Equal(t, "foo", winget.Name)
}
