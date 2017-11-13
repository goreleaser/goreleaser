package brew

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
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

	bts, err := ioutil.ReadFile("testdata/test.rb")
	assert.NoError(t, err)
	// ioutil.WriteFile("testdata/test.rb", []byte(formulae), 0644)

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
		Version: "1.0.1",
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
	ctx.AddBinary("darwinamd64", "bin", "bin", path)
	client := &DummyClient{}
	assert.Error(t, doRun(ctx, client))
	assert.False(t, client.CreatedFile)

	_, err = os.Create(path)
	assert.NoError(t, err)
	assert.NoError(t, doRun(ctx, client))
	assert.True(t, client.CreatedFile)

	bts, err := ioutil.ReadFile("testdata/run_pipe.rb")
	assert.NoError(t, err)
	// ioutil.WriteFile("testdata/run_pipe.rb", []byte(client.Content), 0644)

	assert.Equal(t, string(bts), client.Content)
}

func TestRunPipeFormatOverride(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var path = filepath.Join(folder, "bin.zip")
	_, err = os.Create(path)
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
			Archive: config.Archive{
				Format: "tar.gz",
				FormatOverrides: []config.FormatOverride{
					{
						Format: "zip",
						Goos:   "darwin",
					},
				},
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
	ctx.AddBinary("darwinamd64", "bin", "bin", path)
	client := &DummyClient{}
	assert.NoError(t, doRun(ctx, client))
	assert.True(t, client.CreatedFile)
	assert.Contains(t, client.Content, "bin.zip")
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
	var ctx = &context.Context{
		Publish: true,
		Config: config.Project{
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
	}
	ctx.AddBinary("darwinamd64", "foo", "bar", "baz")
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
