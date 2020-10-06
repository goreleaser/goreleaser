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
nfpms:
  - homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.NoError(t, err)
	assert.Equal(t, "http://goreleaser.github.io", prop.NFPMs[0].Homepage, "yaml did not load correctly")
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
	assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 2: field invalid_yaml not found in type config.Build")
}

func TestInvalidYaml(t *testing.T) {
	_, err := Load("testdata/invalid.yml")
	assert.EqualError(t, err, "yaml: line 1: did not find expected node content")
}

func TestConfigWithAnchors(t *testing.T) {
	_, err := Load("testdata/anchor.yaml")
	assert.NoError(t, err)
}

func TestConfigWithEnvTemplates(t *testing.T) {
	os.Setenv("GORELEASER_TEST_BASE_PATH", "/path/to/darkside")
	defer os.Unsetenv("GORELEASER_TEST_BASE_PATH")

	project, err := Load("testdata/config_with_env.yml")
	assert.Equal(t, "BASE_PATH=/path/to/darkside", project.Env[0])

	assert.Equal(t, "CGO_ENABLED=0", project.Builds[0].Env[0])
	assert.Equal(t, "PKG_CONFIG_PATH=/path/to/darkside/config", project.Builds[0].Env[1])

	assert.NoError(t, err)
}

func TestConfigWithEnvTemplatesMissingEnv(t *testing.T) {

	_, err := Load("testdata/config_with_env.yml")

	assert.Error(t, err, "Should return error when Env Variable is not set")
}
