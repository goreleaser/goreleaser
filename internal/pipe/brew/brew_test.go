package brew

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNameWithDash(t *testing.T) {
	require.Equal(t, "SomeBinary", formulaNameFor("some-binary"))
}

func TestNameWithUnderline(t *testing.T) {
	require.Equal(t, "SomeBinary", formulaNameFor("some_binary"))
}

func TestNameWithDots(t *testing.T) {
	require.Equal(t, "Binaryv000", formulaNameFor("binaryv0.0.0"))
}

func TestNameWithAT(t *testing.T) {
	require.Equal(t, "SomeBinaryAT1", formulaNameFor("some_binary@1"))
}

func TestSimpleName(t *testing.T) {
	require.Equal(t, "Binary", formulaNameFor("binary"))
}

var defaultTemplateData = templateData{
	Desc:     "Some desc",
	Homepage: "https://google.com",
	LinuxPackages: []releasePackage{
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			OS:          "linux",
			Arch:        "amd64",
			Install:     []string{`bin.install "test"`},
		},
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm6.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			OS:          "linux",
			Arch:        "arm",
			Install:     []string{`bin.install "test"`},
		},
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			OS:          "linux",
			Arch:        "arm64",
			Install:     []string{`bin.install "test"`},
		},
	},
	MacOSPackages: []releasePackage{
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
			OS:          "darwin",
			Arch:        "amd64",
			Install:     []string{`bin.install "test"`},
		},
		{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_arm64.tar.gz",
			SHA256:      "1df5fdc2bad4ed4c28fbdc77b6c542988c0dc0e2ae34e0dc912bbb1c66646c58",
			OS:          "darwin",
			Arch:        "arm64",
			Install:     []string{`bin.install "test"`},
		},
	},
	Name:                 "Test",
	Version:              "0.1.3",
	Caveats:              []string{},
	HasOnlyAmd64MacOsPkg: false,
}

func assertDefaultTemplateData(t *testing.T, formulae string) {
	t.Helper()
	require.Contains(t, formulae, "class Test < Formula")
	require.Contains(t, formulae, `homepage "https://google.com"`)
	require.Contains(t, formulae, `url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`)
	require.Contains(t, formulae, `sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`)
	require.Contains(t, formulae, `version "0.1.3"`)
}

func TestFullFormulae(t *testing.T) {
	data := defaultTemplateData
	data.License = "MIT"
	data.Caveats = []string{"Here are some caveats"}
	data.Dependencies = []config.HomebrewDependency{{Name: "gtk+"}}
	data.Conflicts = []string{"svn"}
	data.Plist = "it works"
	data.PostInstall = []string{`touch "/tmp/foo"`, `system "echo", "done"`}
	data.CustomBlock = []string{"devel do", `  url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`, `  sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`, "end"}
	data.Tests = []string{`system "#{bin}/{{.ProjectName}}", "-version"`}
	formulae, err := doBuildFormula(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFullFormulaeLinuxOnly(t *testing.T) {
	data := defaultTemplateData
	data.MacOSPackages = []releasePackage{}
	formulae, err := doBuildFormula(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFullFormulaeMacOSOnly(t *testing.T) {
	data := defaultTemplateData
	data.LinuxPackages = []releasePackage{}
	formulae, err := doBuildFormula(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFormulaeSimple(t *testing.T) {
	formulae, err := doBuildFormula(testctx.NewWithCfg(config.Project{}), defaultTemplateData)
	require.NoError(t, err)
	assertDefaultTemplateData(t, formulae)
	require.NotContains(t, formulae, "def caveats")
	require.NotContains(t, formulae, "def plist;")
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
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"git_remote": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.Brews[0].Repository = config.RepoRef{
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
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.Brews[0].Repository = config.RepoRef{
					Owner:  "test",
					Name:   "test",
					Branch: "update-{{.Version}}",
					PullRequest: config.PullRequest{
						Enabled: true,
					},
				}
			},
		},
		"custom_download_strategy": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].DownloadStrategy = "GitHubPrivateRepositoryReleaseDownloadStrategy"
			},
		},
		"custom_require": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].DownloadStrategy = "CustomDownloadStrategy"
				ctx.Config.Brews[0].CustomRequire = "custom_download_strategy"
			},
		},
		"custom_block": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].CustomBlock = `head "https://github.com/caarlos0/test.git"`
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishErrorAs: &tmpl.Error{},
		},
		"valid_repository_templates": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Env = map[string]string{
					"FOO": "templated",
				}
				ctx.Config.Brews[0].Repository.Owner = "{{.Env.FOO}}"
				ctx.Config.Brews[0].Repository.Name = "{{.Env.FOO}}"
			},
		},
		"invalid_repository_name_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "{{ .Asdsa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid_repository_owner_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Repository.Owner = "{{ .Asdsa }"
				ctx.Config.Brews[0].Repository.Name = "test"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid_repository_skip_upload_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].SkipUpload = "{{ .Asdsa }"
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
		"invalid_install_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Repository.Owner = "test"
				ctx.Config.Brews[0].Repository.Name = "test"
				ctx.Config.Brews[0].Install = "{{ .aaaa }"
			},
			expectedRunErrorAs: &tmpl.Error{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Brews: []config.Homebrew{
						{
							Name: name,
							IDs: []string{
								"foo",
							},
							Description: "Run pipe test formula and FOO={{ .Env.FOO }}",
							Caveats:     "don't do this {{ .ProjectName }}",
							Test:        "system \"true\"\nsystem \"#{bin}/foo\", \"-h\"",
							Plist:       `<xml>whatever</xml>`,
							Dependencies: []config.HomebrewDependency{
								{Name: "zsh", Type: "optional"},
								{Name: "bash", Version: "3.2.57"},
								{Name: "fish", Type: "optional", Version: "v1.2.3"},
								{Name: "powershell", Type: "optional", OS: "mac"},
								{Name: "ash", Version: "1.0.0", OS: "linux"},
							},
							Conflicts:   []string{"gtk+", "qt"},
							Service:     "run foo/bar\nkeep_alive true",
							PostInstall: "system \"echo\"\ntouch \"/tmp/hi\"",
							Install:     `bin.install "{{ .ProjectName }}_{{.Os}}_{{.Arch}} => {{.ProjectName}}"`,
							Goamd64:     "v1",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.WithVersion("1.0.1"),
				testctx.WithCurrentTag("v1.0.1"),
			)
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bar_bin.tar.gz",
				Path:    "doesnt matter",
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "bar",
					artifact.ExtraFormat: "tar.gz",
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tar.gz",
				Path:    path,
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "foo",
					artifact.ExtraFormat: "tar.gz",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.tar.gz",
				Path:   path,
				Goos:   "darwin",
				Goarch: "arm64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "foo",
					artifact.ExtraFormat: "tar.gz",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tar.gz",
				Path:    path,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "foo",
					artifact.ExtraFormat: "tar.gz",
				},
			})

			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			client := client.NewMock()
			distFile := filepath.Join(folder, "homebrew", name+".rb")

			require.NoError(t, Pipe{}.Default(ctx))

			err = runAll(ctx, client)
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
			if url := ctx.Config.Brews[0].Repository.Git.URL; url == "" {
				require.True(t, client.CreatedFile, "should have created a file")
			} else {
				content = testlib.CatFileFromBareRepositoryOnBranch(
					t, url,
					ctx.Config.Brews[0].Repository.Branch,
					name+".rb",
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
			Brews: []config.Homebrew{
				{
					Name:        "foo_{{ .Env.FOO_BAR }}",
					Description: "Foo bar",
					Homepage:    "https://goreleaser.com",
					Goamd64:     "v1",
					Install:     `bin.install "foo"`,
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
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
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
			Brews: []config.Homebrew{
				{
					Name:    "foo",
					Goamd64: "v1",
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
					Name:    "bar",
					Goamd64: "v1",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
				{
					Name:    "foobar",
					Goamd64: "v1",
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
					Name:    "baz",
					Goamd64: "v1",
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
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	cli := client.NewMock()
	require.NoError(t, runAll(ctx, cli))
	require.EqualError(t, publishAll(ctx, cli), `brew.skip_upload is set`)
	require.True(t, cli.CreatedFile)

	for _, brew := range ctx.Config.Brews {
		distFile := filepath.Join(folder, "homebrew", brew.Name+".rb")
		_, err := os.Stat(distFile)
		require.NoError(t, err, "file should exist: "+distFile)
	}
}

func TestRunPipeForMultipleAmd64Versions(t *testing.T) {
	for name, fn := range map[string]func(ctx *context.Context){
		"v1": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goamd64 = "v1"
		},
		"v2": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goamd64 = "v2"
		},
		"v3": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goamd64 = "v3"
		},
		"v4": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goamd64 = "v4"
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Brews: []config.Homebrew{
						{
							Name:        name,
							Description: "Run pipe test formula",
							Repository: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Homepage:     "https://github.com/goreleaser",
							Install:      `bin.install "foo"`,
							ExtraInstall: `man1.install "./man/foo.1.gz"`,
						},
					},
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.GitHubTokenType,
				testctx.WithVersion("1.0.1"),
				testctx.WithCurrentTag("v1.0.1"),
			)
			fn(ctx)
			for _, a := range []struct {
				name    string
				goos    string
				goarch  string
				goamd64 string
			}{
				{
					name:   "bin",
					goos:   "darwin",
					goarch: "arm64",
				},
				{
					name:   "arm64",
					goos:   "linux",
					goarch: "arm64",
				},
				{
					name:    "amd64v2",
					goos:    "linux",
					goarch:  "amd64",
					goamd64: "v1",
				},
				{
					name:    "amd64v2",
					goos:    "linux",
					goarch:  "amd64",
					goamd64: "v2",
				},
				{
					name:    "amd64v3",
					goos:    "linux",
					goarch:  "amd64",
					goamd64: "v3",
				},
				{
					name:    "amd64v3",
					goos:    "linux",
					goarch:  "amd64",
					goamd64: "v4",
				},
			} {
				path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    fmt.Sprintf("%s.tar.gz", a.name),
					Path:    path,
					Goos:    a.goos,
					Goarch:  a.goarch,
					Goamd64: a.goamd64,
					Type:    artifact.UploadableArchive,
					Extra: map[string]interface{}{
						artifact.ExtraID:     a.name,
						artifact.ExtraFormat: "tar.gz",
					},
				})
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			client := client.NewMock()
			distFile := filepath.Join(folder, "homebrew", name+".rb")

			require.NoError(t, runAll(ctx, client))
			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualRb(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeForMultipleArmVersions(t *testing.T) {
	for name, fn := range map[string]func(ctx *context.Context){
		"multiple_armv5": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goarm = "5"
		},
		"multiple_armv6": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goarm = "6"
		},
		"multiple_armv7": func(ctx *context.Context) {
			ctx.Config.Brews[0].Goarm = "7"
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					Brews: []config.Homebrew{
						{
							Name:         name,
							Description:  "Run pipe test formula and FOO={{ .Env.FOO }}",
							Caveats:      "don't do this {{ .ProjectName }}",
							Test:         "system \"true\"\nsystem \"#{bin}/foo\", \"-h\"",
							Plist:        `<xml>whatever</xml>`,
							Dependencies: []config.HomebrewDependency{{Name: "zsh"}, {Name: "bash", Type: "recommended"}},
							Conflicts:    []string{"gtk+", "qt"},
							Install:      `bin.install "{{ .ProjectName }}"`,
							Repository: config.RepoRef{
								Owner: "test",
								Name:  "test",
							},
							Homepage: "https://github.com/goreleaser",
						},
					},
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.GitHubTokenType,
				testctx.WithVersion("1.0.1"),
				testctx.WithCurrentTag("v1.0.1"),
			)
			fn(ctx)
			for _, a := range []struct {
				name   string
				goos   string
				goarch string
				goarm  string
			}{
				{
					name:   "bin",
					goos:   "darwin",
					goarch: "amd64",
				},
				{
					name:   "arm64",
					goos:   "linux",
					goarch: "arm64",
				},
				{
					name:   "armv5",
					goos:   "linux",
					goarch: "arm",
					goarm:  "5",
				},
				{
					name:   "armv6",
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					name:   "armv7",
					goos:   "linux",
					goarch: "arm",
					goarm:  "7",
				},
			} {
				path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   fmt.Sprintf("%s.tar.gz", a.name),
					Path:   path,
					Goos:   a.goos,
					Goarch: a.goarch,
					Goarm:  a.goarm,
					Type:   artifact.UploadableArchive,
					Extra: map[string]interface{}{
						artifact.ExtraID:     a.name,
						artifact.ExtraFormat: "tar.gz",
					},
				})
				f, err := os.Create(path)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			client := client.NewMock()
			distFile := filepath.Join(folder, "homebrew", name+".rb")

			require.NoError(t, runAll(ctx, client))
			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualRb(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Brews: []config.Homebrew{
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
		ids:     []string{"foo"},
		goarm:   "6",
		goamd64: "v1",
	}.Error())
	require.False(t, client.CreatedFile)
}

func TestRunPipeMultipleArchivesSameOsBuild(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Brews: []config.Homebrew{
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
			goarm  string
		}
	}{
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
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
				goarm  string
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
				goarm  string
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
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
			},
		},
		{
			expectedError: ErrMultipleArchivesSameOS,
			osarchs: []struct {
				goos   string
				goarch string
				goarm  string
			}{
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "5",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "6",
				},
				{
					goos:   "linux",
					goarch: "arm",
					goarm:  "7",
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
				Extra: map[string]interface{}{
					artifact.ExtraID:     fmt.Sprintf("foo%d", idx),
					artifact.ExtraFormat: "tar.gz",
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
			Brews: []config.Homebrew{
				{
					Name:        "foo",
					Homepage:    "https://goreleaser.com",
					Description: "Fake desc",
					Repository: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					ExtraInstall: `man1.install "./man/foo.1.gz"`,
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
		Extra: map[string]interface{}{
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
			Brews: []config.Homebrew{
				{
					Name:         "foo",
					Homepage:     "https://goreleaser.com",
					Description:  "Fake desc",
					ExtraInstall: `man1.install "./man/foo.1.gz"`,
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
		Extra: map[string]interface{}{
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
	golden.RequireEqualRb(t, []byte(client.Content))
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Brews: []config.Homebrew{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				Goamd64: "v1",
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
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
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
		ctx.Config.Brews[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload true set by template", func(t *testing.T) {
		ctx.Config.Brews[0].SkipUpload = "{{.Env.SKIP_UPLOAD}}"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.Brews[0].SkipUpload = "auto"
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
		Brews: []config.Homebrew{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
				Goamd64: "v1",
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
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
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
		Brews: []config.Homebrew{
			{
				Plist: "<xml>... whatever</xml>",
				Tap:   repo,
			},
		},
	}, testctx.GitHubTokenType)
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Brews[0].Name)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitMessageTemplate)
	require.Equal(t, repo, ctx.Config.Brews[0].Repository)
	require.True(t, ctx.Deprecated)
}

func TestGHFolder(t *testing.T) {
	require.Equal(t, "bar.rb", buildFormulaPath("", "bar.rb"))
	require.Equal(t, "fooo/bar.rb", buildFormulaPath("fooo", "bar.rb"))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Brews: []config.Homebrew{
				{},
			},
		}, testctx.Skip(skips.Homebrew))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Brews: []config.Homebrew{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Brews: []config.Homebrew{{}},
	})

	client := client.NewMock()
	testlib.AssertSkipped(t, runAll(ctx, client))
}

func TestInstalls(t *testing.T) {
	t.Run("provided", func(t *testing.T) {
		install, err := installs(
			testctx.New(),
			config.Homebrew{Install: "bin.install \"foo\"\nbin.install \"bar\""},
			&artifact.Artifact{},
		)
		require.NoError(t, err)
		require.Equal(t, []string{
			`bin.install "foo"`,
			`bin.install "bar"`,
		}, install)
	})

	t.Run("from archives", func(t *testing.T) {
		install, err := installs(
			testctx.New(),
			config.Homebrew{},
			&artifact.Artifact{
				Type: artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraBinaries: []string{"foo", "bar"},
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, []string{
			`bin.install "bar"`,
			`bin.install "foo"`,
		}, install)
	})

	t.Run("from binary", func(t *testing.T) {
		install, err := installs(
			testctx.New(),
			config.Homebrew{},
			&artifact.Artifact{
				Name: "foo_macos",
				Type: artifact.UploadableBinary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "foo",
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, []string{
			`bin.install "foo_macos" => "foo"`,
		}, install)
	})

	t.Run("from template", func(t *testing.T) {
		install, err := installs(
			testctx.New(),
			config.Homebrew{
				Install: `bin.install "foo_{{.Os}}" => "foo"`,
			},
			&artifact.Artifact{
				Name: "foo_darwin",
				Goos: "darwin",
				Type: artifact.UploadableBinary,
			},
		)
		require.NoError(t, err)
		require.Equal(t, []string{
			`bin.install "foo_darwin" => "foo"`,
		}, install)
	})
}

func TestRunPipeUniversalBinary(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Brews: []config.Homebrew{
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
					Install: `bin.install "unibin"`,
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
		Extra: map[string]interface{}{
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
			Brews: []config.Homebrew{
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
					Install: `bin.install "unibin"`,
					Goamd64: "v1",
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
		Extra: map[string]interface{}{
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
		Extra: map[string]interface{}{
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
		Extra: map[string]interface{}{
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
