package brew

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNameWithDash(t *testing.T) {
	require.Equal(t, formulaNameFor("some-binary"), "SomeBinary")
}

func TestNameWithUnderline(t *testing.T) {
	require.Equal(t, formulaNameFor("some_binary"), "SomeBinary")
}

func TestNameWithDots(t *testing.T) {
	require.Equal(t, formulaNameFor("binaryv0.0.0"), "Binaryv0_0_0")
}

func TestNameWithAT(t *testing.T) {
	require.Equal(t, formulaNameFor("some_binary@1"), "SomeBinaryAT1")
}

func TestSimpleName(t *testing.T) {
	require.Equal(t, formulaNameFor("binary"), "Binary")
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
			SHA256:      "1633f61598ab0791e213135923624eb342196b349490sadasdsadsadasdasdsd",
			OS:          "darwin",
			Arch:        "arm64",
			Install:     []string{`bin.install "test"`},
		},
	},
	Name:    "Test",
	Version: "0.1.3",
	Caveats: []string{},
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
	data.PostInstall = `system "touch", "/tmp/foo"`
	data.CustomBlock = []string{"devel do", `  url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`, `  sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`, "end"}
	data.Tests = []string{`system "#{bin}/{{.ProjectName}} -version"`}
	formulae, err := doBuildFormula(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFullFormulaeLinuxOnly(t *testing.T) {
	data := defaultTemplateData
	data.MacOSPackages = []releasePackage{}
	formulae, err := doBuildFormula(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFullFormulaeMacOSOnly(t *testing.T) {
	data := defaultTemplateData
	data.LinuxPackages = []releasePackage{}
	formulae, err := doBuildFormula(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqualRb(t, []byte(formulae))
}

func TestFormulaeSimple(t *testing.T) {
	formulae, err := doBuildFormula(context.New(config.Project{}), defaultTemplateData)
	require.NoError(t, err)
	assertDefaultTemplateData(t, formulae)
	require.NotContains(t, formulae, "def caveats")
	require.NotContains(t, formulae, "def plist;")
}

func TestSplit(t *testing.T) {
	parts := split("system \"true\"\nsystem \"#{bin}/foo -h\"")
	require.Equal(t, []string{"system \"true\"", "system \"#{bin}/foo -h\""}, parts)
	parts = split("")
	require.Equal(t, []string{}, parts)
	parts = split("\n  ")
	require.Equal(t, []string{}, parts)
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare              func(ctx *context.Context)
		expectedRunError     string
		expectedPublishError string
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"custom_download_strategy": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].DownloadStrategy = "GitHubPrivateRepositoryReleaseDownloadStrategy"
			},
		},
		"custom_require": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].DownloadStrategy = "CustomDownloadStrategy"
				ctx.Config.Brews[0].CustomRequire = "custom_download_strategy"
			},
		},
		"custom_block": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

				ctx.Config.Brews[0].CustomBlock = `head "https://github.com/caarlos0/test.git"`
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
				ctx.Config.Brews[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"valid_tap_templates": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Env = map[string]string{
					"FOO": "templated",
				}
				ctx.Config.Brews[0].Tap.Owner = "{{.Env.FOO}}"
				ctx.Config.Brews[0].Tap.Name = "{{.Env.FOO}}"
			},
		},
		"invalid_tap_name_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "{{ .Asdsa }"
			},
			expectedRunError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid_tap_owner_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].Tap.Owner = "{{ .Asdsa }"
				ctx.Config.Brews[0].Tap.Name = "test"
			},
			expectedRunError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid_tap_skip_upload_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.Brews[0].SkipUpload = "{{ .Asdsa }"
				ctx.Config.Brews[0].Tap.Owner = "test"
				ctx.Config.Brews[0].Tap.Name = "test"
			},
			expectedRunError: `template: tmpl:1: unexpected "}" in operand`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			ctx := &context.Context{
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Env: map[string]string{
					"FOO": "foo_is_bar",
				},
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					Brews: []config.Homebrew{
						{
							Name: name,
							IDs: []string{
								"foo",
							},
							Description:  "A run pipe test formula and FOO={{ .Env.FOO }}",
							Caveats:      "don't do this {{ .ProjectName }}",
							Test:         "system \"true\"\nsystem \"#{bin}/foo -h\"",
							Plist:        `<xml>whatever</xml>`,
							Dependencies: []config.HomebrewDependency{{Name: "zsh", Type: "optional"}, {Name: "bash"}},
							Conflicts:    []string{"gtk+", "qt"},
							Install:      `bin.install "{{ .ProjectName }}"`,
						},
					},
				},
			}
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bar_bin.tar.gz",
				Path:   "doesnt matter",
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "bar",
					artifact.ExtraFormat: "tar.gz",
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.tar.gz",
				Path:   path,
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:     "foo",
					artifact.ExtraFormat: "tar.gz",
				},
			})

			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			client := client.NewMock()
			distFile := filepath.Join(folder, name+".rb")

			if tt.expectedRunError == "" {
				require.NoError(t, runAll(ctx, client))
			} else {
				require.EqualError(t, runAll(ctx, client), tt.expectedRunError)
				return
			}
			if tt.expectedPublishError != "" {
				require.EqualError(t, publishAll(ctx, client), tt.expectedPublishError)
				return
			}

			require.NoError(t, publishAll(ctx, client))
			require.True(t, client.CreatedFile)
			golden.RequireEqualRb(t, []byte(client.Content))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)
			require.Equal(t, client.Content, string(distBts))
		})
	}
}

func TestRunPipeNameTemplate(t *testing.T) {
	folder := t.TempDir()
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Env: map[string]string{
			"FOO_BAR": "is_bar",
		},
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Brews: []config.Homebrew{
				{
					Name: "foo_{{ .Env.FOO_BAR }}",
					Tap: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
			},
		},
	}
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "tar.gz",
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "foo_is_bar.rb")

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
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Env: map[string]string{
			"FOO_BAR":     "is_bar",
			"SKIP_UPLOAD": "true",
		},
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Brews: []config.Homebrew{
				{
					Name: "foo",
					Tap: config.RepoRef{
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
					Tap: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
				},
				{
					Name: "foobar",
					Tap: config.RepoRef{
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
					Tap: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
					IDs: []string{
						"foo",
					},
					SkipUpload: "{{ .Env.SKIP_UPLOAD }}",
				},
			},
		},
	}
	path := filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
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
		distFile := filepath.Join(folder, brew.Name+".rb")
		_, err := os.Stat(distFile)
		require.NoError(t, err, "file should exist: "+distFile)
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
			ctx := &context.Context{
				TokenType: context.TokenTypeGitHub,
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Env: map[string]string{
					"FOO": "foo_is_bar",
				},
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					Brews: []config.Homebrew{
						{
							Name:         name,
							Description:  "A run pipe test formula and FOO={{ .Env.FOO }}",
							Caveats:      "don't do this {{ .ProjectName }}",
							Test:         "system \"true\"\nsystem \"#{bin}/foo -h\"",
							Plist:        `<xml>whatever</xml>`,
							Dependencies: []config.HomebrewDependency{{Name: "zsh"}, {Name: "bash", Type: "recommended"}},
							Conflicts:    []string{"gtk+", "qt"},
							Install:      `bin.install "{{ .ProjectName }}"`,
							Tap: config.RepoRef{
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
				},
			}
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
			distFile := filepath.Join(folder, name+".rb")

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
	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			Brews: []config.Homebrew{
				{
					Tap: config.RepoRef{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	}
	client := client.NewMock()
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeMultipleArchivesSameOsBuild(t *testing.T) {
	ctx := context.New(
		config.Project{
			Brews: []config.Homebrew{
				{
					Tap: config.RepoRef{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	)

	ctx.TokenType = context.TokenTypeGitHub
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
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.1",
		},
		Version:   "1.2.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Brews: []config.Homebrew{
				{
					Name: "foo",
					Tap: config.RepoRef{
						Owner: "foo",
						Name:  "bar",
					},
				},
			},
		},
	}

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

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Brews: []config.Homebrew{
			{
				Tap: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.Env = map[string]string{
		"SKIP_UPLOAD": "true",
	}
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
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
	// TODO: skip when ctx.Config.Release.Draft=true ?
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Brews: []config.Homebrew{
			{
				Tap: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
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

	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "myproject",
			Brews: []config.Homebrew{
				{},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.Brews[0].Name)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Name)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Email)
	require.NotEmpty(t, ctx.Config.Brews[0].CommitMessageTemplate)
}

func TestGHFolder(t *testing.T) {
	require.Equal(t, "bar.rb", buildFormulaPath("", "bar.rb"))
	require.Equal(t, "fooo/bar.rb", buildFormulaPath("fooo", "bar.rb"))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Brews: []config.Homebrew{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := context.New(config.Project{
		Brews: []config.Homebrew{{}},
	})

	client := client.NewMock()
	testlib.AssertSkipped(t, runAll(ctx, client))
}

func TestInstalls(t *testing.T) {
	t.Run("provided", func(t *testing.T) {
		require.Equal(t, []string{
			`bin.install "foo"`,
			`bin.install "bar"`,
		}, installs(
			config.Homebrew{Install: "bin.install \"foo\"\nbin.install \"bar\""},
			&artifact.Artifact{},
		))
	})

	t.Run("from archives", func(t *testing.T) {
		require.Equal(t, []string{
			`bin.install "bar"`,
			`bin.install "foo"`,
		}, installs(
			config.Homebrew{},
			&artifact.Artifact{
				Type: artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraBinaries: []string{"foo", "bar"},
				},
			},
		))
	})

	t.Run("from binary", func(t *testing.T) {
		require.Equal(t, []string{
			`bin.install "foo_macos" => "foo"`,
		}, installs(
			config.Homebrew{},
			&artifact.Artifact{
				Name: "foo_macos",
				Type: artifact.UploadableBinary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "foo",
				},
			},
		))
	})
}

func TestRunPipeUniversalBinary(t *testing.T) {
	folder := t.TempDir()
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			Dist:        folder,
			ProjectName: "unibin",
			Brews: []config.Homebrew{
				{
					Name: "unibin",
					Tap: config.RepoRef{
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
	}
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
		},
	})

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	client := client.NewMock()
	distFile := filepath.Join(folder, "unibin.rb")

	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))
	require.True(t, client.CreatedFile)
	golden.RequireEqualRb(t, []byte(client.Content))
	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)
	require.Equal(t, client.Content, string(distBts))
}
