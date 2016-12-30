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

var testFormulaeExpected = `class Test < Formula
  desc "Some desc"
  homepage "https://google.com"
  url "https://github.com/caarlos0/test/releases/download/v0.1.3/test_#{%x(uname -s).gsub(/\n/, '')}_#{%x(uname -m).gsub(/\n/, '')}.tar.gz"
  head "https://github.com/caarlos0/test.git"
  version "v0.1.3"

  def install
    bin.install "test"
  end

  def caveats
    "Here are some caveats"
  end
end
`

func TestFormulae(t *testing.T) {
	assert := assert.New(t)
	out, err := doBuildFormulae(templateData{
		BinaryName: "test",
		Desc:       "Some desc",
		Homepage:   "https://google.com",
		Name:       "Test",
		Repo:       "caarlos0/test",
		Tag:        "v0.1.3",
		Caveats:    "Here are some caveats",
	})
	assert.NoError(err)
	assert.NoError(err)
	assert.Equal(testFormulaeExpected, out.String())
}
