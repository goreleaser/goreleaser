package config

import (
	"bytes"
	"fmt"
	"os"
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
	conf := `
nfpms:
  - homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	require.NoError(t, err)
	require.Equal(t, "http://goreleaser.github.io", prop.NFPMs[0].Homepage, "yaml did not load correctly")
}

type errorReader struct{}

func (errorReader) Read(_ []byte) (n int, err error) {
	return 1, fmt.Errorf("error")
}

func TestLoadBadReader(t *testing.T) {
	_, err := LoadReader(errorReader{})
	require.Error(t, err)
}

func TestFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config")
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

func TestVersion(t *testing.T) {
	t.Run("allow no version", func(t *testing.T) {
		_, err := LoadReader(bytes.NewReader(nil))
		require.NoError(t, err)
	})
	t.Run("allow v0", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 0"))
		require.NoError(t, err)
	})
	t.Run("allow v1", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 1"))
		require.NoError(t, err)
	})
	t.Run("do not allow v2", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 2"))
		require.Error(t, err)
		require.ErrorIs(t, err, VersionError{2})
	})
}
