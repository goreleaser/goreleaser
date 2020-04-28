package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestConfigFlagNotSetButExists(t *testing.T) {
	for _, name := range []string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			folder, back := setup(t)
			defer back()
			require.NoError(t, os.Rename(
				filepath.Join(folder, "goreleaser.yml"),
				filepath.Join(folder, name),
			))
			proj, err := loadConfig("")
			require.NoError(t, err)
			require.NotEqual(t, config.Project{}, proj)
		})
	}
}

func TestConfigFileDoesntExist(t *testing.T) {
	folder, back := setup(t)
	defer back()
	err := os.Remove(filepath.Join(folder, "goreleaser.yml"))
	require.NoError(t, err)
	proj, err := loadConfig("")
	require.NoError(t, err)
	require.Equal(t, config.Project{}, proj)
}
