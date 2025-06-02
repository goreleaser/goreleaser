package cask

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNameWithDash(t *testing.T) {
	require.Equal(t, "somebinary", caskNameFor("SomeBinary"))
}

func TestNameNumberThenWord(t *testing.T) {
	require.Equal(t, "baton-1password", caskNameFor("Baton 1password"))
}

func TestNameWithUnderline(t *testing.T) {
	require.Equal(t, "some_binary", caskNameFor("somE_binarY"))
}

// TODO:
// func TestNameWithDots(t *testing.T) {
// 	require.Equal(t, "Binaryv000", caskNameFor("binaryv0.0.0"))
// }

func TestNameWithAT(t *testing.T) {
	require.Equal(t, "some-binary@1", caskNameFor("some binary@1"))
}

func TestSimpleName(t *testing.T) {
	require.Equal(t, "binary", caskNameFor("binary"))
}

var defaultTemplateData = templateData{
	HomebrewCask: config.HomebrewCask{
		Description: "Some desc",
		Homepage:    "https://google.com",
		Binary:      "mybin",
		Completions: config.HomebrewCaskCompletions{
			Fish: "mybin.fish",
			Bash: "mybin.bash",
			Zsh:  "mybin.zsh",
		},
		Manpage: "mybin.1.gz",
	},
	Name:                 "test",
	Version:              "0.1.3",
	HasOnlyAmd64MacOsPkg: false,
	LinuxPackages: []releasePackage{
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			OS:          "linux",
			Arch:        "amd64",
		},
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			OS:          "linux",
			Arch:        "arm64",
		},
	},
	MacOSPackages: []releasePackage{
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
			OS:          "darwin",
			Arch:        "amd64",
		},
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_arm64.tar.gz",
			SHA256:      "1df5fdc2bad4ed4c28fbdc77b6c542988c0dc0e2ae34e0dc912bbb1c66646c58",
			OS:          "darwin",
			Arch:        "arm64",
		},
	},
}

func assertDefaultTemplateData(t *testing.T, cask string) {
	t.Helper()
	require.Contains(t, cask, "cask \"test\" do")
	require.Contains(t, cask, `homepage "https://google.com"`)
	require.Contains(t, cask, `url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`)
	require.Contains(t, cask, `sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`)
	require.Contains(t, cask, `version "0.1.3"`)
}

func TestFullCask(t *testing.T) {
	data := defaultTemplateData
	data.License = "MIT"
	data.Caveats = "Here are some caveats"
	data.Dependencies = []config.HomebrewCaskDependency{
		{Formula: "goreleaser"},
		{Cask: "goreleaser"},
	}
	data.Conflicts = []config.HomebrewCaskConflict{
		{Formula: "goreleaser"},
		{Cask: "goreleaser"},
	}
	data.Hooks = config.HomebrewCaskHooks{
		Pre: config.HomebrewCaskHook{
			Install:   "pre-install",
			Uninstall: "pre-uninstall",
		},
		Post: config.HomebrewCaskHook{
			Install:   "post-install",
			Uninstall: "post-uninstall",
		},
	}
	data.CustomBlock = `# A custom block
# This particular case is just a comment.`
	cask, err := doBuildCask(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(cask))
}

func TestFullCaskLinuxOnly(t *testing.T) {
	data := defaultTemplateData
	data.MacOSPackages = []releasePackage{}
	cask, err := doBuildCask(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(cask))
}

func TestFullCaskMacOSOnly(t *testing.T) {
	data := defaultTemplateData
	data.LinuxPackages = []releasePackage{}
	cask, err := doBuildCask(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(cask))
}

func TestCaskSimple(t *testing.T) {
	cask, err := doBuildCask(testctx.NewWithCfg(config.Project{}), defaultTemplateData)
	require.NoError(t, err)
	assertDefaultTemplateData(t, cask)
	require.NotContains(t, cask, "def caveats")
}

func TestSplit(t *testing.T) {
	parts := split("system \"true\"\nsystem \"#{bin}/foo\", \"-h\"")
	require.Equal(t, []string{"system \"true\"", "system \"#{bin}/foo\", \"-h\""}, parts)
	parts = split("")
	require.Equal(t, []string{}, parts)
	parts = split("\n  ")
	require.Equal(t, []string{}, parts)
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare                func(ctx *context.Context)
		expectedRunError       string
		expectedRunErrorAs     any
		expectedPublishError   string
		expectedPublishErrorAs any
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"git_remote": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Casks[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.Casks[0].Repository = config.RepoRef{
					Name:   "test",
					Branch: "main",
					Git: config.GitRepoRef{
						URL:        testlib.GitMakeBareRepository(t),
						PrivateKey: testlib.MakeNewSSHKey(t, ""),
					},
				}
			},
		},
		"open_pr": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Casks[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.Casks[0].Repository = config.RepoRef{
					Owner:  "test",
					Name:   "test",
					Branch: "update-{{.Version}}",
					PullRequest: config.PullRequest{
						Enabled: true,
					},
				}
			},
		},
		"custom_block": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Casks[0].CustomBlock = `head "https://github.com/caarlos0/test.git"`
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishErrorAs: &tmpl.Error{},
		},
		"valid_repository_templates": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Env = map[string]string{
					"FOO": "templated",
				}
				ctx.Config.Casks[0].Repository.Owner = "{{.Env.FOO}}"
				ctx.Config.Casks[0].Repository.Name = "{{.Env.FOO}}"
			},
		},
		"invalid_repository_name_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid_repository_owner_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "{{ .Asdsa }"
				ctx.Config.Casks[0].Repository.Name = "test"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid_repository_skip_upload_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].SkipUpload = "{{ .Asdsa }"
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"uninstall": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Uninstall = config.HomebrewCaskUninstall{
					Launchctl: []string{"launchctl1", "launchctl2", "launchctl3"},
					Quit:      []string{"quit1", "quit2", "quit3"},
					LoginItem: []string{"loginitem1", "loginitem2", "loginitem3"},
					Trash:     []string{"trash1", "trash2", "trash3"},
					Delete:    []string{"delete1", "delete2", "delete3"},
				}
			},
		},
		"zap": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Zap = config.HomebrewCaskUninstall{
					Launchctl: []string{"launchctl1", "launchctl2", "launchctl3"},
					Quit:      []string{"quit1", "quit2", "quit3"},
					LoginItem: []string{"loginitem1", "loginitem2", "loginitem3"},
					Trash:     []string{"trash1", "trash2", "trash3"},
					Delete:    []string{"delete1", "delete2", "delete3"},
				}
			},
		},
		"url_parameters_curl": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].URLAdditional.Using = ":homebrew_curl"
				ctx.Config.Casks[0].URLAdditional.Cookies = map[string]string{"license": "accept"}
				ctx.Config.Casks[0].URLAdditional.Referer = "https://example-url-parameters.com/"
				ctx.Config.Casks[0].URLAdditional.Headers = []string{"Accept: application/octet-stream"}
				ctx.Config.Casks[0].URLAdditional.UserAgent = "GoReleaser"
			},
		},
		"url_parameters_post": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Casks[0].Repository.Owner = "test"
				ctx.Config.Casks[0].Repository.Name = "test"
				ctx.Config.Casks[0].Homepage = "https://dummyhost-url-parameters.com/"
				ctx.Config.Casks[0].URLAdditional.Using = ":post"
				ctx.Config.Casks[0].URLAdditional.Verified = "https://dummyhost/download/"
				ctx.Config.Casks[0].URLAdditional.Headers = []string{"Accept: application/octet-stream"}
				ctx.Config.Casks[0].URLAdditional.Data = map[string]string{"payload": "hello_world"}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Casks: []config.HomebrewCask{
						{
							Name: name,
							IDs: []string{
								"foo",
							},
							Description: "Run pipe test formula and FOO={{ .Env.FOO }}",
							Caveats:     "don't do this {{ .ProjectName }}",
							Dependencies: []config.HomebrewCaskDependency{
								{Formula: "zsh"},
								{Formula: "bash"},
								{Cask: "fish"},
								{Cask: "powershell"},
								{Formula: "ash"},
							},
							Conflicts: []config.HomebrewCaskConflict{
								{Formula: "bash"},
								{Cask: "fish"},
							},
							Service: "foo.plist",
							Hooks: config.HomebrewCaskHooks{
								Post: config.HomebrewCaskHook{
									Install: "system \"echo\"\ntouch \"/tmp/hi\"",
								},
							},
							Binary: "{{.ProjectName}}",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.WithVersion("1.0.1"),
				testctx.WithCurrentTag("v1.0.1"),
			)
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tgz",
				Path:    filepath.Join(folder, "bin.tgz"),
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tgz",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.txz",
				Path:   filepath.Join(folder, "bin.txz"),
				Goos:   "darwin",
				Goarch: "arm64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "txz",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tar.zst",
				Path:    filepath.Join(folder, "bin.tar.zst"),
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tar.zst",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
			for _, a := range ctx.Artifacts.List() {
				f, err := os.Create(a.Path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bar_bin.tzst",
				Path:    "doesnt matter",
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       "bar",
					artifact.ExtraFormat:   "tzst",
					artifact.ExtraBinaries: []string{"bar"},
				},
			})

			client := client.NewMock()
			distFile := filepath.Join(folder, "homebrew", "Casks", name+".rb")

			require.NoError(t, Pipe{}.Default(ctx))

			err := runAll(ctx, client)
			if tt.expectedRunError != "" {
				require.EqualError(t, err, tt.expectedRunError)
				return
			}
			if tt.expectedRunErrorAs != nil {
				require.ErrorAs(t, err, tt.expectedRunErrorAs)
				return
			}
			require.NoError(t, err)

			err = publishAll(ctx, client)
			if tt.expectedPublishError != "" {
				require.EqualError(t, err, tt.expectedPublishError)
				return
			}
			if tt.expectedPublishErrorAs != nil {
				require.ErrorAs(t, err, tt.expectedPublishErrorAs)
				return
			}
			require.NoError(t, err)

			content := []byte(client.Content)
			if url := ctx.Config.Casks[0].Repository.Git.URL; url == "" {
				require.True(t, client.CreatedFile, "should have created a file")
			} else {
				content = testlib.CatFileFromBareRepositoryOnBranch(
					t, url,
					ctx.Config.Casks[0].Repository.Branch,
					"Casks/"+name+".rb",
				)
			}

			golden.RequireEqualRb(t, content)

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, string(content), string(distBts))
		})
	}
}

func TestRunPipeNameTemplate(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Casks: []config.HomebrewCask{
				{
					Name:        "foo_{{ .Env.FOO_BAR }}",
					Description: "Foo bar",
					Homepage:    "https://goreleaser.com",
					Binary:      "foo",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
			},
			Env: []string{"FOO_BAR=is_bar"},
		},
		testctx.WithVersion("1.0.1"),
		testctx.WithCurrentTag("v1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "homebrew", "foo_is_bar.rb")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualRb(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeMultipleBrewsWithSkip(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Casks: []config.HomebrewCask{
				{
					Name: "foo",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
				{
					Name: "bar",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
				{
					Name: "foobar",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "true",
				},
				{
					Name: "baz",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "{{ .Env.SKIP_UPLOAD }}",
				},
			},
			Env: []string{
				"FOO_BAR=is_bar",
				"SKIP_UPLOAD=true",
			},
		},
		testctx.WithVersion("1.0.1"),
		testctx.WithCurrentTag("v1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cli := client.NewMock()
	require.NoError(t, runAll(ctx, cli))
	require.EqualError(t, publishAll(ctx, cli), `brew.skip_upload is set`)
	require.True(t, cli.CreatedFile)

	for _, brew := range ctx.Config.Casks {
		distFile := filepath.Join(folder, "homebrew", brew.Name+".rb")
		_, err := os.Stat(distFile)
		require.NoError(t, err, "file should exist: "+distFile)
	}
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Casks: []config.HomebrewCask{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				IDs: []string{"foo"},
			},
		},
	}, testctx.GitHubTokenType)
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, runAll(ctx, client), ErrNoArchivesFound{
		ids: []string{"foo"},
	}.Error())
	require.False(t, client.CreatedFile)
}

func TestRunPipeMultipleArchivesSameOsBuild(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Casks: []config.HomebrewCask{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.GitHubTokenType)

	f, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

	tests := []struct {
		expectedError error
		osarchs       []struct {
			goos   string
			goarch string
		}
	}{
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
			}{
				{
					goos:   "darwin",
					goarch: "amd64",
				},
				{
					goos:   "darwin",
					goarch: "amd64",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
			}{
				{
					goos:   "linux",
					goarch: "amd64",
				},
				{
					goos:   "linux",
					goarch: "amd64",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
			}{
				{
					goos:   "linux",
					goarch: "arm64",
				},
				{
					goos:   "linux",
					goarch: "arm64",
				},
			},
		},
	}

	for _, test := range tests {
		for idx, ttt := range test.osarchs {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   fmt.Sprintf("bin%d", idx),
				Path:   f.Name(),
				Goos:   ttt.goos,
				Goarch: ttt.goarch,
				Type:   artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID:       fmt.Sprintf("foo%d", idx),
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
		}
		client := client.NewMock()
		require.Equal(t, test.expectedError, runAll(ctx, client))
		require.False(t, client.CreatedFile)
		// clean the artifacts for the next run
		ctx.Artifacts = artifact.New()
	}
}

func TestRunPipeBinaryRelease(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Casks: []config.HomebrewCask{
				{
					Name:        "foo",
					Homepage:    "https://goreleaser.com",
					Description: "Fake desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					Binary:  "foo",
					Manpage: "./man/foo.1.gz",
				},
			},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)
	path := filepath.Join(folder, "dist/foo_darwin_all/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_macos",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableBinary,
		Extra: map[string]any{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "binary",
			artifact.ExtraBinary: "foo",
		},
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualRb(t, []byte(client.Content))
}

func TestRunPipePullRequest(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Casks: []config.HomebrewCask{
				{
					Name:        "foo",
					Homepage:    "https://goreleaser.com",
					Description: "Fake desc",
					Manpage:     "./man/foo.1.gz",
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
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
	)
	path := filepath.Join(folder, "dist/foo_darwin_all/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_macos",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableBinary,
		Extra: map[string]any{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "binary",
			artifact.ExtraBinary: "foo",
		},
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	require.True(t, client.OpenedPullRequest)
	require.True(t, client.SyncedFork)
	golden.RequireEqualRb(t, []byte(client.Content))
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Casks: []config.HomebrewCask{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
		Env: []string{"SKIP_UPLOAD=true"},
	}, testctx.WithCurrentTag("v1.0.1"), testctx.GitHubTokenType)
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})
	client := client.NewMock()

	assertNoPublish := func(t *testing.T) {
		t.Helper()
		require.NoError(t, runAll(ctx, client))
		testlib.AssertSkipped(t, publishAll(ctx, client))
		require.False(t, client.CreatedFile)
	}
	t.Run("skip upload true", func(t *testing.T) {
		ctx.Config.Casks[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload true set by template", func(t *testing.T) {
		ctx.Config.Casks[0].SkipUpload = "{{.Env.SKIP_UPLOAD}}"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.Casks[0].SkipUpload = "auto"
		ctx.Semver.Prerelease = "beta1"
		assertNoPublish(t)
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Casks: []config.HomebrewCask{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.WithCurrentTag("v1.0.0"))
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})
	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
}

func TestDefault(t *testing.T) {
	testlib.Mktmp(t)
	repo := config.RepoRef{
		Owner:  "owner",
		Name:   "name",
		Token:  "aaa",
		Branch: "feat",
		Git: config.GitRepoRef{
			URL:        "git@github.com:foo/bar",
			SSHCommand: "ssh ",
			PrivateKey: "/fake",
		},
		PullRequest: config.PullRequest{
			Enabled: true,
			Base: config.PullRequestBase{
				Owner:  "foo2",
				Name:   "bar2",
				Branch: "branch2",
			},
			Draft: true,
		},
	}
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "myproject",
		Casks: []config.HomebrewCask{
			{
				Repository: repo,
			},
		},
	}, testctx.GitHubTokenType)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Casks[0].Name)
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Casks[0].Binary)
	require.NotEmpty(t, ctx.Config.Casks[0].CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Casks[0].CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Casks[0].CommitMessageTemplate)
	require.Equal(t, repo, ctx.Config.Casks[0].Repository)
}

func TestGHFolder(t *testing.T) {
	require.Equal(t, "bar.rb", buildCaskPath("", "bar.rb"))
	require.Equal(t, "fooo/bar.rb", buildCaskPath("fooo", "bar.rb"))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Casks: []config.HomebrewCask{
				{},
			},
		}, testctx.Skip(skips.Homebrew))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Casks: []config.HomebrewCask{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Casks: []config.HomebrewCask{{}},
	})

	client := client.NewMock()
	testlib.AssertSkipped(t, runAll(ctx, client))
}

func TestRunPipeUniversalBinary(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Casks: []config.HomebrewCask{
				{
					Name:        "unibin",
					Homepage:    "https://goreleaser.com",
					Description: "Fake desc",
					Repository: config.RepoRef{
						Owner: "unibin",
						Name:  "bar",
					},
					IDs: []string{
						"unibin",
					},
					Binary: "unibin",
				},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
			artifact.ExtraReplaces: true,
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "homebrew", "unibin.rb")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualRb(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}

func TestRunPipeUniversalBinaryNotReplacing(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Casks: []config.HomebrewCask{
				{
					Name:        "unibin",
					Homepage:    "https://goreleaser.com",
					Description: "Fake desc",
					Repository: config.RepoRef{
						Owner: "unibin",
						Name:  "bar",
					},
					IDs: []string{
						"unibin",
					},
					Binary: "unibin",
				},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin_amd64.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "bin_arm64.tar.gz",
		Path:    path,
		Goos:    "darwin",
		Goarch:  "arm64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "all",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "unibin",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"unibin"},
			artifact.ExtraReplaces: false,
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "homebrew", "unibin.rb")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualRb(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}
