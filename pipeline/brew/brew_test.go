package brew

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/client"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
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
	Binary:   "test",
	Desc:     "Some desc",
	Homepage: "https://google.com",
	Name:     "Test",
	Repo: config.Repo{
		Owner: "caarlos0",
		Name:  "test",
	},
	Tag:     "v0.1.3",
	Version: "0.1.3",
	File:    "test_Darwin_x86_64",
	SHA256:  "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68",
	Format:  "tar.gz",
}

func assertDefaultTemplateData(t *testing.T, formulae string) {
	assert := assert.New(t)
	assert.Contains(formulae, "class Test < Formula")
	assert.Contains(formulae, "homepage \"https://google.com\"")
	assert.Contains(formulae, "url \"https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz\"")
	assert.Contains(formulae, "sha256 \"1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c68\"")
	assert.Contains(formulae, "version \"0.1.3\"")
}

func TestFullFormulae(t *testing.T) {
	assert := assert.New(t)
	data := defaultTemplateData
	data.Caveats = "Here are some caveats"
	data.Dependencies = []string{"gtk", "git"}
	data.Conflicts = []string{"conflicting_dep"}
	data.Plist = "it works"
	data.Install = []string{"custom install script", "another install script"}
	out, err := doBuildFormula(data)
	assert.NoError(err)
	formulae := out.String()
	assertDefaultTemplateData(t, formulae)
	assert.Contains(formulae, "def caveats")
	assert.Contains(formulae, "Here are some caveats")
	assert.Contains(formulae, "depends_on \"gtk\"")
	assert.Contains(formulae, "depends_on \"git\"")
	assert.Contains(formulae, "conflicts_with \"conflicting_dep\"")
	assert.Contains(formulae, "custom install script")
	assert.Contains(formulae, "another install script")
	assert.Contains(formulae, "def plist;")
}

func TestFormulaeSimple(t *testing.T) {
	assert := assert.New(t)
	out, err := doBuildFormula(defaultTemplateData)
	assert.NoError(err)
	formulae := out.String()
	assertDefaultTemplateData(t, formulae)
	assert.NotContains(formulae, "def caveats")
	assert.NotContains(formulae, "depends_on")
	assert.NotContains(formulae, "def plist;")
}

func TestRunPipe(t *testing.T) {
	assert := assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	_, err = os.Create(filepath.Join(folder, "bin.tar.gz"))
	assert.NoError(err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
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
		Archives: map[string]string{
			"darwinamd64": "bin",
		},
	}
	client := &DummyClient{}
	assert.NoError(doRun(ctx, client))
	assert.True(client.CreatedFile)
}

func TestRunPipeBrewNotSetup(t *testing.T) {
	assert := assert.New(t)
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
	assert.Equal(ErrNoDarwin64Build, doRun(ctx, client))
	assert.False(client.CreatedFile)
}

func TestRunPipeNoDarwinBuild(t *testing.T) {
	assert := assert.New(t)
	var ctx = &context.Context{}
	client := &DummyClient{}
	assert.NoError(doRun(ctx, client))
	assert.False(client.CreatedFile)
}

type DummyClient struct {
	CreatedFile bool
}

func (client *DummyClient) GetInfo(ctx *context.Context) (info client.Info, err error) {
	return
}

func (client *DummyClient) CreateRelease(ctx *context.Context) (releaseID int, err error) {
	return
}

func (client *DummyClient) CreateFile(ctx *context.Context, content bytes.Buffer, path string) (err error) {
	client.CreatedFile = true
	return
}

func (client *DummyClient) Upload(ctx *context.Context, releaseID int, name string, file *os.File) (err error) {
	return
}
