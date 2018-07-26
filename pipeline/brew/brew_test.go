package brew

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
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
	Desc:        "Some desc",
	Homepage:    "https://google.com",
	DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz",
	Name:        "Test",
	Version:     "0.1.3",
	Caveats:     []string{},
	SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
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
	data.Install = []string{"custom install script", "another install script"}
	data.Tests = []string{`system "#{bin}/foo -version"`}
	out, err := doBuildFormula(data)
	assert.NoError(t, err)
	formulae := out.String()

	var golden = "testdata/test.rb.golden"
	if *update {
		ioutil.WriteFile(golden, []byte(formulae), 0655)
	}
	bts, err := ioutil.ReadFile(golden)
	assert.NoError(t, err)
	assert.Equal(t, string(bts), formulae)
}

func TestFormulaeSimple(t *testing.T) {
	out, err := doBuildFormula(defaultTemplateData)
	assert.NoError(t, err)
	formulae := out.String()
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
		"default": func(ctx *context.Context) {},
		"github_enterprise_url": func(ctx *context.Context) {
			ctx.Config.GitHubURLs.Download = "http://github.example.org"
		},
		"custom_download_strategy": func(ctx *context.Context) {
			ctx.Config.Brew.DownloadStrategy = "GitHubPrivateRepositoryReleaseDownloadStrategy"
		},
		"binary_overriden": func(ctx *context.Context) {
			ctx.Config.Archive.Format = "binary"
			ctx.Config.Archive.FormatOverrides = []config.FormatOverride{
				{
					Goos:   "darwin",
					Format: "zip",
				},
			}
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
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					GitHubURLs: config.GitHubURLs{
						Download: "https://github.com",
					},
					Archive: config.Archive{
						Format: "tar.gz",
					},
					Release: config.Release{
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
					},
					Brew: config.Homebrew{
						Name: name,
						GitHub: config.Repo{
							Owner: "test",
							Name:  "test",
						},
						Description:  "A run pipe test formula",
						Homepage:     "https://github.com/goreleaser",
						Caveats:      "don't do this",
						Test:         "system \"true\"\nsystem \"#{bin}/foo -h\"",
						Plist:        `<xml>whatever</xml>`,
						Dependencies: []string{"zsh", "bash"},
						Conflicts:    []string{"gtk+", "qt"},
						Install:      `bin.install "foo"`,
					},
				},
			}
			fn(ctx)
			var format = getFormat(ctx)
			var path = filepath.Join(folder, "bin."+format)
			ctx.Artifacts.Add(artifact.Artifact{
				Name:   "bin." + format,
				Path:   path,
				Goos:   "darwin",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
			})

			_, err = os.Create(path)
			assert.NoError(t, err)
			client := &DummyClient{}
			var distFile = filepath.Join(folder, name+".rb")

			assert.NoError(t, doRun(ctx, client))
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

func TestRunPipeNoDarwin64Build(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				Format: "tar.gz",
			},
			Brew: config.Homebrew{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}
	client := &DummyClient{}
	assert.Equal(t, ErrNoDarwin64Build, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeMultipleDarwin64Build(t *testing.T) {
	var ctx = context.New(
		config.Project{
			Archive: config.Archive{
				Format: "tar.gz",
			},
			Brew: config.Homebrew{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	)
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "bin1",
		Path:   "doesnt mather",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "bin2",
		Path:   "doesnt mather",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})
	client := &DummyClient{}
	assert.Equal(t, ErrTooManyDarwin64Builds, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeBrewNotSetup(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeBinaryRelease(t *testing.T) {
	var ctx = context.New(
		config.Project{
			Archive: config.Archive{
				Format: "binary",
			},
			Brew: config.Homebrew{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	)
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "bin",
		Path:   "doesnt mather",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Brew: config.Homebrew{
			GitHub: config.Repo{
				Owner: "test",
				Name:  "test",
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	var path = filepath.Join(folder, "whatever.tar.gz")
	_, err = os.Create(path)
	assert.NoError(t, err)
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})
	client := &DummyClient{}

	var assertNoPublish = func(t *testing.T) {
		testlib.AssertSkipped(t, doRun(ctx, client))
		assert.False(t, client.CreatedFile)
	}
	t.Run("skip upload", func(tt *testing.T) {
		ctx.Config.Release.Draft = false
		ctx.Config.Brew.SkipUpload = true
		ctx.SkipPublish = false
		assertNoPublish(tt)
	})
	t.Run("skip publish", func(tt *testing.T) {
		ctx.Config.Release.Draft = false
		ctx.Config.Brew.SkipUpload = false
		ctx.SkipPublish = true
		assertNoPublish(tt)
	})
	t.Run("draft release", func(tt *testing.T) {
		ctx.Config.Release.Draft = true
		ctx.Config.Brew.SkipUpload = false
		ctx.SkipPublish = false
		assertNoPublish(tt)
	})
}

func TestRunPipeFormatBinary(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				Format: "binary",
			},
		},
	}
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestDefault(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Config: config.Project{
			ProjectName: "myproject",
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
	assert.Equal(t, ctx.Config.ProjectName, ctx.Config.Brew.Name)
	assert.NotEmpty(t, ctx.Config.Brew.CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Brew.CommitAuthor.Email)
	assert.Equal(t, `bin.install "foo"`, ctx.Config.Brew.Install)
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int64, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content bytes.Buffer, path, msg string) (err error) {
	client.CreatedFile = true
	bts, _ := ioutil.ReadAll(&content)
	client.Content = string(bts)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error) {
	return
}
