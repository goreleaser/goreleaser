package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestConfigFlagNotSetButExists(t *testing.T) {
	for _, name := range []string{
		".config/goreleaser.yml",
		".config/goreleaser.yaml",
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			folder := setup(t)
			require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
			require.NoError(t, os.Rename(
				filepath.Join(folder, "goreleaser.yml"),
				filepath.Join(folder, name),
			))
			proj, err := loadConfig(true, "")
			require.NoError(t, err)
			require.NotEqual(t, config.Project{}, proj)
		})
	}
}

var proConfig = `version: 2
pro: true
some_possibly_pro_option: {}
`

func TestProConfigFile(t *testing.T) {
	folder := setup(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(folder, "goreleaser.yml"),
		[]byte(proConfig),
		0o644,
	))
	t.Run("strict", func(t *testing.T) {
		proj, err := loadConfig(true, "goreleaser.yml")
		require.Error(t, err)
		require.Equal(t, config.Project{
			Version: 2,
			Pro:     true,
		}, proj)
	})

	t.Run("relaxed", func(t *testing.T) {
		proj, err := loadConfig(false, "goreleaser.yml")
		require.NoError(t, err)
		require.Equal(t, config.Project{
			Version: 2,
			Pro:     true,
		}, proj)
	})
}

func TestConfigFileDoesntExist(t *testing.T) {
	folder := setup(t)
	err := os.Remove(filepath.Join(folder, "goreleaser.yml"))
	require.NoError(t, err)
	proj, err := loadConfig(true, "")
	require.NoError(t, err)
	require.Equal(t, config.Project{}, proj)
}

func TestConfigFileFromStdin(t *testing.T) {
	folder := setup(t)
	err := os.Remove(filepath.Join(folder, "goreleaser.yml"))
	require.NoError(t, err)
	proj, err := loadConfig(true, "-")
	require.NoError(t, err)
	require.Equal(t, config.Project{}, proj)
}
