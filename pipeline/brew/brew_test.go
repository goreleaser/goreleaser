package brew

import (
	"bytes"
	"flag"
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
	DownloadURL: "https://github.com",
	Name:        "Test",
	Repo: config.Repo{
		Owner: "caarlos0",
		Name:  "test",
	},
	Tag:     "v0.1.3",
	Version: "0.1.3",
	File:    "test_Darwin_x86_64.tar.gz",
	SHA256:  "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
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
	data.Caveats = "Here are some caveats"
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
}

func TestRunPipe(t *testing.T) {
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
			ProjectName: "run-pipe",
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
		Publish: true,
	}
	var path = filepath.Join(folder, "bin.tar.gz")
	ctx.Artifacts.Add(artifact.Artifact{
		Name:   "bin.tar.gz",
		Path:   path,
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})
	client := &DummyClient{}
	assert.Error(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)

	_, err = os.Create(path)
	assert.NoError(t, err)

	t.Run("default git url", func(tt *testing.T) {
		assert.NoError(tt, doRun(ctx, client))
		assert.True(tt, client.CreatedFile)
		var golden = "testdata/run_pipe.rb.golden"
		if *update {
			ioutil.WriteFile(golden, []byte(client.Content), 0655)
		}
		bts, err := ioutil.ReadFile(golden)
		assert.NoError(tt, err)
		assert.Equal(tt, string(bts), client.Content)
	})

	t.Run("github enterprise url", func(tt *testing.T) {
		ctx.Config.GitHubURLs.Download = "http://github.example.org"
		assert.NoError(tt, doRun(ctx, client))
		assert.True(tt, client.CreatedFile)
		var golden = "testdata/run_pipe_enterprise.rb.golden"
		if *update {
			ioutil.WriteFile(golden, []byte(client.Content), 0644)
		}
		bts, err := ioutil.ReadFile(golden)
		assert.NoError(tt, err)
		assert.Equal(tt, string(bts), client.Content)
	})
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
		Publish: true,
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
	ctx.Publish = true
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
		Config:  config.Project{},
		Publish: true,
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
	ctx.Publish = true
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

func TestRunPipeNoPublish(t *testing.T) {
	var ctx = &context.Context{
		Publish: false,
	}
	client := &DummyClient{}
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
}

func TestRunPipeDraftRelease(t *testing.T) {
	var ctx = &context.Context{
		Publish: true,
		Config: config.Project{
			Release: config.Release{
				Draft: true,
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
	testlib.AssertSkipped(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)
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
	assert.NotEmpty(t, ctx.Config.Brew.CommitAuthor.Name)
	assert.NotEmpty(t, ctx.Config.Brew.CommitAuthor.Email)
	assert.Equal(t, `bin.install "foo"`, ctx.Config.Brew.Install)
}

type DummyClient struct {
	CreatedFile bool
	Content     string
}

func (client *DummyClient) CreateRelease(ctx *context.Context, body string) (releaseID int, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error) {
	client.CreatedFile = true
	bts, _ := ioutil.ReadAll(&content)
	client.Content = string(bts)
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error) {
	return
}
