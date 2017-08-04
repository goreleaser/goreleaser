package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	var assert = assert.New(t)
	r := Repo{Owner: "goreleaser", Name: "godownloader"}
	assert.Equal("goreleaser/godownloader", r.String(), "not equal")
}

func TestLoadReader(t *testing.T) {
	var conf = `
fpm:
  homepage: http://goreleaser.github.io
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

func TestInvalidFields(t *testing.T) {
	var assert = assert.New(t)
	_, err := Load("testdata/invalid_config.yml")
	assert.EqualError(err, "unknown fields in the config file: invalid_root, archive.invalid_archive, archive.format_overrides[0].invalid_archive_fmtoverrides, brew.invalid_brew, brew.github.invalid_brew_github, builds[0].invalid_builds, builds[0].hooks.invalid_builds_hooks, builds[0].ignored_builds[0].invalid_builds_ignore, fpm.invalid_fpm, release.invalid_release, release.github.invalid_release_github, build.invalid_build, builds.hooks.invalid_build_hook, builds.ignored_builds[0].invalid_build_ignore, snapshot.invalid_snapshot")
}

func TestInvalidYaml(t *testing.T) {
	var assert = assert.New(t)
	_, err := Load("testdata/invalid.yml")
	assert.EqualError(err, "yaml: line 1: did not find expected node content")
}
