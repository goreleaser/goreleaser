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
	assert.Equal(
		t,
		"goreleaser/godownloader",
		Repo{Owner: "goreleaser", Name: "godownloader"}.String(),
	)
}

func TestEmptyRepoNameAndOwner(t *testing.T) {
	assert.Empty(t, Repo{}.String())
}

func TestLoadReader(t *testing.T) {
	var conf = `
fpm:
  homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.NoError(t, err)
	assert.Equal(t, "http://goreleaser.github.io", prop.FPM.Homepage, "yaml did not load correctly")
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 1, fmt.Errorf("error")
}
func TestLoadBadReader(t *testing.T) {
	_, err := LoadReader(errorReader{})
	assert.Error(t, err)
}

func TestFile(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "config")
	assert.NoError(t, err)
	_, err = Load(filepath.Join(f.Name()))
	assert.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	_, err := Load("/nope/no-way.yml")
	assert.Error(t, err)
}

func TestInvalidFields(t *testing.T) {
	_, err := Load("testdata/invalid_config.yml")
	assert.EqualError(t, err, "unknown fields in the config file: invalid_root, archive.invalid_archive, archive.format_overrides[0].invalid_archive_fmtoverrides, brew.invalid_brew, brew.github.invalid_brew_github, builds[0].invalid_builds, builds[0].hooks.invalid_builds_hooks, builds[0].ignored_builds[0].invalid_builds_ignore, fpm.invalid_fpm, release.invalid_release, release.github.invalid_release_github, build.invalid_build, builds.hooks.invalid_build_hook, builds.ignored_builds[0].invalid_build_ignore, snapshot.invalid_snapshot, docker[0].invalid_docker, changelog.invalid_changelog, changelog.filters.invalid_filters")
}

func TestInvalidYaml(t *testing.T) {
	_, err := Load("testdata/invalid.yml")
	assert.EqualError(t, err, "yaml: line 1: did not find expected node content")
}
