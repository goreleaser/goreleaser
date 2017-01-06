package brew

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	BinaryName: "test",
	Desc:       "Some desc",
	Homepage:   "https://google.com",
	Name:       "Test",
	Repo:       "caarlos0/test",
	Tag:        "v0.1.3",
}

func assertDefaultTemplateData(t *testing.T, formulae string) {
	assert := assert.New(t)
	assert.Contains(formulae, "class Test < Formula")
	assert.Contains(formulae, "homepage \"https://google.com\"")
	assert.Contains(formulae, "url \"https://github.com/caarlos0/test/releases/download/v0.1.3/test_Darwin_x86_64.tar.gz\"")
	assert.Contains(formulae, "head \"https://github.com/caarlos0/test.git\"")
	assert.Contains(formulae, "version \"v0.1.3\"")
	assert.Contains(formulae, "bin.install \"test\"")
}

func TestFullFormulae(t *testing.T) {
	assert := assert.New(t)
	data := defaultTemplateData
	data.Caveats = "Here are some caveats"
	out, err := doBuildFormulae(data)
	assert.NoError(err)
	formulae := out.String()
	assertDefaultTemplateData(t, formulae)
	assert.Contains(formulae, "def caveats")
	assert.Contains(formulae, "Here are some caveats")
}

func TestFormulaeNoCaveats(t *testing.T) {
	assert := assert.New(t)
	out, err := doBuildFormulae(defaultTemplateData)
	assert.NoError(err)
	formulae := out.String()
	assertDefaultTemplateData(t, formulae)
	assert.NotContains(formulae, "def caveats")
}
