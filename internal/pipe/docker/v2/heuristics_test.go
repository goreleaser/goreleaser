package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindRootProjectExtraFiles(t *testing.T) {
	tests := []struct {
		name       string
		extraFiles []string
		expected   []string
	}{
		{
			name:       "no extra files",
			extraFiles: nil,
			expected:   nil,
		},
		{
			name:       "config files only",
			extraFiles: []string{"config.yaml", "templates/index.html"},
			expected:   nil,
		},
		{
			name:       "go.mod and go.sum",
			extraFiles: []string{"go.mod", "go.sum"},
			expected:   []string{"go.mod"},
		},
		{
			name:       "Cargo.toml",
			extraFiles: []string{"Cargo.toml"},
			expected:   []string{"Cargo.toml"},
		},
		{
			name:       "pyproject.toml",
			extraFiles: []string{"pyproject.toml"},
			expected:   []string{"pyproject.toml"},
		},
		{
			name:       "nested path extracts base",
			extraFiles: []string{"subdir/go.mod"},
			expected:   []string{"go.mod"},
		},
		{
			name:       "mixed files",
			extraFiles: []string{"config.yaml", "go.mod", "README.md", "Cargo.toml"},
			expected:   []string{"go.mod", "Cargo.toml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findRootProjectExtraFiles(tt.extraFiles)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFindRootProjectExtraFiles_EnvDisable(t *testing.T) {
	t.Setenv("GORELEASER_NO_SLOW_DOCKER_WARN", "1")
	result := findRootProjectExtraFiles([]string{"go.mod", "Cargo.toml"})
	require.Nil(t, result)
}

func TestEmitExtraFilesWarning(t *testing.T) {
	emitExtraFilesWarning([]string{"go.mod", "go.sum"})
}
