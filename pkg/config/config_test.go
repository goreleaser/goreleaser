package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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

func TestChangelogAIBaseURL(t *testing.T) {
	t.Run("with base_url", func(t *testing.T) {
		project, err := LoadReader(strings.NewReader(`
version: 2
changelog:
  ai:
    use: openai
    model: gpt-4o
    base_url: https://api.example.com/v1
`))
		require.NoError(t, err)
		require.Equal(t, "openai", project.Changelog.AI.Use)
		require.Equal(t, "gpt-4o", project.Changelog.AI.Model)
		require.Equal(t, "https://api.example.com/v1", project.Changelog.AI.BaseURL)
	})

	t.Run("without base_url", func(t *testing.T) {
		project, err := LoadReader(strings.NewReader(`
version: 2
changelog:
  ai:
    use: openai
    model: gpt-4o
`))
		require.NoError(t, err)
		require.Equal(t, "openai", project.Changelog.AI.Use)
		require.Empty(t, project.Changelog.AI.BaseURL)
	})

	t.Run("empty ai block", func(t *testing.T) {
		project, err := LoadReader(strings.NewReader(`
version: 2
changelog:
  sort: asc
`))
		require.NoError(t, err)
		require.Empty(t, project.Changelog.AI.Use)
		require.Empty(t, project.Changelog.AI.BaseURL)
	})
}

func TestArchiveZstdConfig(t *testing.T) {
	project, err := LoadReader(strings.NewReader(`
version: 2
archives:
  - formats: [tar.zst]
    zstd:
      level: best
      window_size: 27
`))
	require.NoError(t, err)
	require.Len(t, project.Archives, 1)
	require.Equal(t, "best", project.Archives[0].Zstd.Level)
	require.NotNil(t, project.Archives[0].Zstd.WindowSize)
	require.Equal(t, 27, *project.Archives[0].Zstd.WindowSize)
}

func TestArrayEmptyVsNil(t *testing.T) {
	conf := `
builds: []
# blobs:
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	require.NoError(t, err)
	require.NotNil(t, prop.Builds)
	require.Empty(t, prop.Builds)
	require.Nil(t, prop.Blobs)
	require.Empty(t, prop.Blobs)
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
	_, err = Load(f.Name())
	require.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	_, err := Load("/nope/no-way.yml")
	require.Error(t, err)
}

func TestInvalidFields(t *testing.T) {
	_, err := Load("testdata/invalid_config.yml")
	require.EqualError(t, err, "yaml: unmarshal errors:\n  line 3: field invalid_yaml not found in type config.Build")
}

func TestInvalidYaml(t *testing.T) {
	_, err := Load("testdata/invalid.yml")
	require.EqualError(t, err, "yaml: line 2: did not find expected node content")
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
	t.Run("do not allow no version with errors", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("nope: nope"))
		require.Error(t, err)
		require.ErrorIs(t, err, VersionError{0})
	})
	t.Run("allow v0", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 0"))
		require.NoError(t, err)
	})
	t.Run("do not allow v0 with errors", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 0\nnope: nope"))
		require.Error(t, err)
		require.ErrorIs(t, err, VersionError{0})
	})
	t.Run("allow v1", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 1"))
		require.NoError(t, err)
	})
	t.Run("do not allow v1 with errors", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 1\nnope: nope"))
		require.Error(t, err)
		require.ErrorIs(t, err, VersionError{1})
	})
	t.Run("allow v2", func(t *testing.T) {
		_, err := LoadReader(strings.NewReader("version: 2\nbuilds: []"))
		require.NoError(t, err)
	})
}

func TestPro(t *testing.T) {
	_, err := LoadReader(strings.NewReader("version: 2\npro: true\nnope: true\n"))
	require.ErrorIs(t, err, ErrProConfig)
}

func TestPrivateKeyOmittedFromJSON(t *testing.T) {
	for name, v := range map[string]any{
		"git":       GitRepoRef{URL: "ssh://example.invalid/repo.git", PrivateKey: "SECRET"},
		"aur":       AUR{PrivateKey: "SECRET"},
		"aursource": AURSource{PrivateKey: "SECRET"},
	} {
		t.Run(name, func(t *testing.T) {
			bts, err := json.Marshal(v)
			require.NoError(t, err)
			require.NotContains(t, string(bts), "SECRET")
			require.NotContains(t, string(bts), "private_key")
		})
	}
}

func TestGitPrivateKeyTemplateJSON(t *testing.T) {
	for _, key := range []string{
		"{{ .Env.PRIVATE_KEY }}",
		"{{ readFile .Env.PRIVATE_KEY_PATH }}",
		"{{ .Env.HOME }}/.ssh/id_ed25519",
	} {
		t.Run(key, func(t *testing.T) {
			ref := GitRepoRef{
				URL:        "ssh://example.invalid/repo.git",
				PrivateKey: key,
			}
			data, err := json.Marshal(ref)
			require.NoError(t, err)
			var restored GitRepoRef
			require.NoError(t, json.Unmarshal(data, &restored))
			require.Equal(t, ref, restored)
		})
	}
}
