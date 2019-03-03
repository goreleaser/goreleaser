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

func TestRepoString(t *testing.T) {
	tests := []struct {
		expected string
		repo     Repo
	}{
		{
			"",
			Repo{},
		},
		{
			"helloworld",
			Repo{Name: "helloworld"},
		},
		{
			"goreleaser/godownloader",
			Repo{Owner: "goreleaser", Name: "godownloader"},
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			assert.Equal(t, test.expected, test.repo.String())
		})
	}
}

func TestLoadReader(t *testing.T) {
	var conf = `
nfpm:
  homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.NoError(t, err)
	assert.Equal(t, "http://goreleaser.github.io", prop.NFPM.Homepage, "yaml did not load correctly")
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
