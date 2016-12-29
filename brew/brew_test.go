package brew

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"os"
	"io/ioutil"
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


func TestFormulae(t *testing.T) {
	assert := assert.New(t)
	out, err := buildFormulae(templateData{
		BinaryName: "test",
		Desc:       "Some desc",
		Homepage:   "https://google.com",
		Name:       "Test",
		Repo:       "caarlos0/test",
		Tag:        "v0.1.3",
		Caveats:    "Here are some caveats",
	})
	assert.NoError(err)
	f, err := os.Open("./brew/test_files/test.txt")
	bts, _ := ioutil.ReadAll(f)
	assert.NoError(err)
	assert.Equal(string(bts), out.String())
}
