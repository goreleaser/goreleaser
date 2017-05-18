package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fmt"

	"github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	var assert = assert.New(t)
	r := Repo{"goreleaser", "godownloader"}
	assert.Equal("goreleaser/godownloader", r.String(), "not equal")
}

func TestLoadReader(t *testing.T) {
	var conf = `
homepage: &homepage http://goreleaser.github.io
fpm:
  homepage: *homepage
`
	var assert = assert.New(t)
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.Nil(err)
	assert.Equal("http://goreleaser.github.io", prop.FPM.Homepage, "yaml did not load correctly")
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 1, fmt.Errorf("error")
}
func TestLoadBadReader(t *testing.T) {
	var assert = assert.New(t)
	_, err := LoadReader(errorReader{})
	assert.Error(err)
}

func TestFile(t *testing.T) {
	var assert = assert.New(t)
	f, err := ioutil.TempFile(os.TempDir(), "config")
	assert.NoError(err)
	_, err = Load(filepath.Join(f.Name()))
	assert.NoError(err)
}

func TestFileNotFound(t *testing.T) {
	var assert = assert.New(t)
	_, err := Load("/nope/no-way.yml")
	assert.Error(err)
}
