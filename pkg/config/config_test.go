package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepo(t *testing.T) {
	require.Equal(
		t,
		"goreleaser/godownloader",
		Repo{Owner: "goreleaser", Name: "godownloader"}.String(),
	)
}

func TestEmptyRepoNameAndOwner(t *testing.T) {
	require.Empty(t, Repo{}.String())
}

func TestLoadReader(t *testing.T) {
	var conf = `
nfpms:
  - homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	require.NoError(t, err)
	require.Equal(t, "http://goreleaser.github.io", prop.NFPMs[0].Homepage, "yaml did not load correctly")
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 1, fmt.Errorf("error")
}
func TestLoadBadReader(t *testing.T) {
	_, err := LoadReader(errorReader{})
	require.Error(t, err)
}

func TestFile(t *testing.T) {
	f, err := ioutil.TempFile(t.TempDir(), "config")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	_, err = Load(filepath.Join(f.Name()))
	require.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	_, err := Load("/nope/no-way.yml")
	require.Error(t, err)
}

func TestInvalidFields(t *testing.T) {
	_, err := Load("testdata/invalid_config.yml")
	require.EqualError(t, err, "yaml: unmarshal errors:\n  line 2: field invalid_yaml not found in type config.Build")
}

func TestInvalidYaml(t *testing.T) {
	_, err := Load("testdata/invalid.yml")
	require.EqualError(t, err, "yaml: line 1: did not find expected node content")
}

func TestConfigWithAnchors(t *testing.T) {
	_, err := Load("testdata/anchor.yaml")
	require.NoError(t, err)
}
