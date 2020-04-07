package brew

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNameWithDash(t *testing.T) {
	assert.Equal(t, formulaNameFor("some-binary"), "SomeBinary")
}

func TestNameWithUnderline(t *testing.T) {
	assert.Equal(t, formulaNameFor("some_binary"), "SomeBinary")
}

func TestSimpleName(t *testing.T) {
	assert.Equal(t, formulaNameFor("binary"), "Binary")
}

var defaultTemplateData = templateData{
	Desc:     "Some desc",
	Homepage: "https://google.com",
	MacOS: downloadable{
		DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
		SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
	},
	Linux: downloadable{
		DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
		SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
	},
	Arm: downloadable{
		DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm6.tar.gz",
		SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
	},
	Arm64: downloadable{
		DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Arm64.tar.gz",
		SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
	},
	Name:    "Test",
	Version: "0.1.3",
	Caveats: []string{},
}

func assertDefaultTemplateData(t *testing.T, formulae string) {
	assert.Contains(t, formulae, "class Test < Formula")
	assert.Contains(t, formulae, `homepage "https://google.com"`)
	assert.Contains(t, formulae, `url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`)
	assert.Contains(t, formulae, `sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`)
	assert.Contains(t, formulae, `version "0.1.3"`)
}

func TestFullFormulae(t *testing.T) {
	data := defaultTemplateData
	data.Caveats = []string{"Here are some caveats"}
	data.Dependencies = []string{"gtk+"}
	data.Conflicts = []string{"svn"}
	data.Plist = "it works"
	data.CustomBlock = []string{"devel do", `  url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz"`, `  sha256 "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68"`, "end"}
	data.Install = []string{"custom install script", "another install script"}
	data.Tests = []string{`system "#{bin}/{{.ProjectName}} -version"`}
	formulae, err := doBuildFormula(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	assert.NoError(t, err)

	var golden = "testdata/test.rb.golden"
	if *update {
		err := ioutil.WriteFile(golden, []byte(formulae), 0655)
		assert.NoError(t, err)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)
	assert.Equal(t, string(bts), formulae)
}

func TestFormulaeSimple(t *testing.T) {
	formulae, err := doBuildFormula(context.New(config.Project{}), defaultTemplateData)
	assert.NoError(t, err)
	assertDefaultTemplateData(t, formulae)
	assert.NotContains(t, formulae, "def caveats")
	assert.NotContains(t, formulae, "depends_on")
	assert.NotContains(t, formulae, "def plist;")
}

func TestSplit(t *testing.T) {
	var parts = split("system \"true\"\nsystem \"#{bin}/foo -h\"")
	assert.Equal(t, []string{"system \"true\"", "system \"#{bin}/foo -h\""}, parts)
	parts = split("")
	assert.Equal(t, []string{}, parts)
	parts = split("\n  ")
	assert.Equal(t, []string{}, parts)
}

func TestRunPipe(t *testing.T) {
	for name, fn := range map[string]func(ctx *context.Context){
		"default": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitHub
			ctx.Config.GitHubURLs.Download = "https://github.com"
			ctx.Config.Release.GitHub.Owner = "test"
			ctx.Config.Release.GitHub.Name = "test"
			ctx.Config.Brews[0].GitHub.Owner = "test"
			ctx.Config.Brews[0].GitHub.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"
		},
		"github_enterprise_url": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitHub
			ctx.Config.GitHubURLs.Download = "https://github.com"
			ctx.Config.Release.GitHub.Owner = "test"
			ctx.Config.Release.GitHub.Name = "test"
			ctx.Config.Brews[0].GitHub.Owner = "test"
			ctx.Config.Brews[0].GitHub.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

			ctx.Config.GitHubURLs.Download = "http://github.example.org"
		},
		"custom_download_strategy": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitHub
			ctx.Config.GitHubURLs.Download = "https://github.com"
			ctx.Config.Release.GitHub.Owner = "test"
			ctx.Config.Release.GitHub.Name = "test"
			ctx.Config.Brews[0].GitHub.Owner = "test"
			ctx.Config.Brews[0].GitHub.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

			ctx.Config.Brews[0].DownloadStrategy = "GitHubPrivateRepositoryReleaseDownloadStrategy"
		},
		"custom_require": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitHub
			ctx.Config.GitHubURLs.Download = "https://github.com"
			ctx.Config.Release.GitHub.Owner = "test"
			ctx.Config.Release.GitHub.Name = "test"
			ctx.Config.Brews[0].GitHub.Owner = "test"
			ctx.Config.Brews[0].GitHub.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

			ctx.Config.Brews[0].DownloadStrategy = "CustomDownloadStrategy"
			ctx.Config.Brews[0].CustomRequire = "custom_download_strategy"
		},
		"custom_block": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitHub
			ctx.Config.GitHubURLs.Download = "https://github.com"
			ctx.Config.Release.GitHub.Owner = "test"
			ctx.Config.Release.GitHub.Name = "test"
			ctx.Config.Brews[0].GitHub.Owner = "test"
			ctx.Config.Brews[0].GitHub.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://github.com/goreleaser"

			ctx.Config.Brews[0].CustomBlock = `head "https://github.com/caarlos0/test.git"`
		},
		"default_gitlab": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitLab
			ctx.Config.GitLabURLs.Download = "https://gitlab.com"
			ctx.Config.Release.GitLab.Owner = "test"
			ctx.Config.Release.GitLab.Name = "test"
			ctx.Config.Brews[0].GitLab.Owner = "test"
			ctx.Config.Brews[0].GitLab.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://gitlab.com/goreleaser"
		},
		"gitlab_enterprise_url": func(ctx *context.Context) {
			ctx.TokenType = context.TokenTypeGitLab
			ctx.Config.GitLabURLs.Download = "https://gitlab.com"
			ctx.Config.Release.GitLab.Owner = "test"
			ctx.Config.Release.GitLab.Name = "test"
			ctx.Config.Brews[0].GitLab.Owner = "test"
			ctx.Config.Brews[0].GitLab.Name = "test"
			ctx.Config.Brews[0].Homepage = "https://gitlab.com/goreleaser"

			ctx.Config.GitLabURLs.Download = "https://gitlab.my-company.org"
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder, err := ioutil.TempDir("", "goreleasertest")
			assert.NoError(t, err)
			var ctx = &context.Context{
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
							Dependencies: []string{"zsh", "bash"},
							Conflicts:    []string{"gtk+", "qt"},
							Install:      `bin.install "{{ .ProjectName }}"`,
						},
					},
				},
			}
			fn(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bar_bin.tar.gz",
				Path:   "doesnt matter",
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":                 "bar",
					"Format":             "tar.gz",
					"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
				},
			})
			var path = filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.tar.gz",
				Path:   path,
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":                 "foo",
					"Format":             "tar.gz",
					"ArtifactUploadHash": "820ead5d9d2266c728dce6d4d55b6460",
				},
			})

			_, err = os.Create(path)
			assert.NoError(t, err)
			client := &DummyClient{}
			var distFile = filepath.Join(folder, name+".rb")

			require.NoError(t, Pipe{}.Run(ctx))
			assert.NoError(t, doPublish(ctx, client))
			assert.True(t, client.CreatedFile)
			var golden = fmt.Sprintf("testdata/%s.rb.golden", name)
			if *update {
				assert.NoError(t, ioutil.WriteFile(golden, []byte(client.Content), 0655))
			}
			bts, err := ioutil.ReadFile(golden)
			assert.NoError(t, err)
			assert.Equal(t, string(bts), client.Content)

			distBts, err := ioutil.ReadFile(distFile)
			assert.NoError(t, err)
			assert.Equal(t, string(bts), string(distBts))
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
		folder, err := ioutil.TempDir("", "goreleasertest")
		assert.NoError(t, err)
		var ctx = &context.Context{
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
						Dependencies: []string{"zsh", "bash"},
						Conflicts:    []string{"gtk+", "qt"},
						Install:      `bin.install "{{ .ProjectName }}"`,
						GitHub: config.Repo{
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
			var path = filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   fmt.Sprintf("%s.tar.gz", a.name),
				Path:   path,
				Goos:   a.goos,
				Goarch: a.goarch,
				Goarm:  a.goarm,
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID":     a.name,
					"Format": "tar.gz",
				},
			})
			_, err := os.Create(path)
			assert.NoError(t, err)
		}

		client := &DummyClient{}
		var distFile = filepath.Join(folder, name+".rb")

		require.NoError(t, Pipe{}.Run(ctx))
		assert.NoError(t, doPublish(ctx, client))
		assert.True(t, client.CreatedFile)
		var golden = fmt.Sprintf("testdata/%s.rb.golden", name)
		if *update {
			assert.NoError(t, ioutil.WriteFile(golden, []byte(client.Content), 0655))
		}
		bts, err := ioutil.ReadFile(golden)
		assert.NoError(t, err)
		assert.Equal(t, string(bts), client.Content)

		distBts, err := ioutil.ReadFile(distFile)
		assert.NoError(t, err)
		assert.Equal(t, string(bts), string(distBts))
	}
}

func TestRunPipeNoDarwin64Build(t *testing.T) {
	var ctx = &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			Brews: []config.Homebrew{
				{
					GitHub: config.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	}
	client := &DummyClient{}
	require.Equal(t, ErrNoArchivesFound, Pipe{}.Run(ctx))
	testlib.AssertSkipped(t, doPublish(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeMultipleArchivesSameOsBuild(t *testing.T) {
	var ctx = context.New(
		config.Project{
			Brews: []config.Homebrew{
				{
					GitHub: config.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	)

	ctx.TokenType = context.TokenTypeGitHub
	f, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer f.Close()

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
			t.Run(ttt.goarch, func(tt *testing.T) {
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   fmt.Sprintf("bin%d", idx),
					Path:   f.Name(),
					Goos:   ttt.goos,
					Goarch: ttt.goarch,
					Type:   artifact.UploadableArchive,
					Extra: map[string]interface{}{
						"ID":     fmt.Sprintf("foo%d", idx),
						"Format": "tar.gz",
					},
				})
			})
		}
		client := &DummyClient{}
		require.Equal(t, test.expectedError, Pipe{}.Run(ctx))
		testlib.AssertSkipped(t, doPublish(ctx, client))
		require.False(t, client.CreatedFile)
		// clean the artifacts for the next run
		ctx.Artifacts = artifact.New()
	}
}

func TestRunPipeBrewNotSetup(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	client := &DummyClient{}
	require.NoError(t, Pipe{}.Run(ctx))
	testlib.AssertSkipped(t, doPublish(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeBinaryRelease(t *testing.T) {
	var ctx = context.New(
		config.Project{
			Brews: []config.Homebrew{
				{
					GitHub: config.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		},
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   "doesnt mather",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})
	client := &DummyClient{}
	require.Equal(t, ErrNoArchivesFound, Pipe{}.Run(ctx))
	testlib.AssertSkipped(t, doPublish(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)

	var newCtx = func() *context.Context {
		var ctx = context.New(config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Release:     config.Release{},
			Brews: []config.Homebrew{
				{
					GitHub: config.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
			},
		})
		ctx.TokenType = context.TokenTypeGitHub
		ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
		var path = filepath.Join(folder, "whatever.tar.gz")
		_, err = os.Create(path)
		assert.NoError(t, err)
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "bin",
			Path:   path,
			Goos:   "darwin",
			Goarch: "amd64",
			Type:   artifact.UploadableArchive,
			Extra: map[string]interface{}{
				"ID":     "foo",
				"Format": "tar.gz",
			},
		})
		return ctx
	}
	var client = &DummyClient{}

	var assertNoPublish = func(t *testing.T, ctx *context.Context) {
		require.NoError(t, Pipe{}.Run(ctx))
		testlib.AssertSkipped(t, doPublish(ctx, client))
		assert.False(t, client.CreatedFile)
	}

	t.Run("skip upload", func(tt *testing.T) {
		var ctx = newCtx()
		ctx.Config.Release.Draft = false
		ctx.Config.Brews[0].SkipUpload = "true"
		ctx.SkipPublish = false
		require.NoError(t, Pipe{}.Run(ctx))
		assertNoPublish(tt, ctx)
	})
	t.Run("skip publish", func(tt *testing.T) {
		var ctx = newCtx()
		ctx.Config.Release.Draft = false
		ctx.Config.Brews[0].SkipUpload = "false"
		ctx.SkipPublish = true
		require.NoError(t, Pipe{}.Run(ctx))
		assertNoPublish(tt, ctx)
	})
}

func TestRunTokenTypeNotImplementedForBrew(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Brews: []config.Homebrew{
			{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	var path = filepath.Join(folder, "whatever.tar.gz")
	_, err = os.Create(path)
	assert.NoError(t, err)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]interface{}{
			"ID":     "foo",
			"Format": "tar.gz",
		},
	})
	client := &DummyClient{}
	require.Equal(t, ErrTokenTypeNotImplementedForBrew, Pipe{}.Run(ctx))
	testlib.AssertSkipped(t, doPublish(ctx, client))
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "myproject",
			Brews: []config.Homebrew{
				{},
			},
			Builds: []config.Build{
				{
					Binary: "foo",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
				},
				{
					Binary: "bar",
					Goos:   []string{"linux", "darwin"},
					Goarch: []string{"386", "amd64"},
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
				{
					Binary: "foobar",
					Goos:   []string{"linux"},
					Goarch: []string{"amd64"},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, ctx.Config.ProjectName, ctx.Config.Brews[0].Name)
	assert.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Brews[0].CommitAuthor.Email)
	assert.Equal(t, `bin.install "foo"`, ctx.Config.Brews[0].Install)
}

func TestGHFolder(t *testing.T) {
	assert.Equal(t, "bar.rb", buildFormulaPath("", "bar.rb"))
	assert.Equal(t, "fooo/bar.rb", buildFormulaPath("fooo", "bar.rb"))
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content []byte, path, msg string) (err error) {
	client.CreatedFile = true
	client.Content = string(content)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return
}
