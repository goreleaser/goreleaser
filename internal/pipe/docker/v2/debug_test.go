package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsFileNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		out      string
		expected bool
	}{
		{
			name:     "empty output",
			out:      "",
			expected: false,
		},
		{
			name:     "unrelated error",
			out:      "ERROR: failed to solve: no such image",
			expected: false,
		},
		{
			name:     "copy directive",
			out:      "ERROR: failed to solve: >>> COPY ./bin /usr/bin",
			expected: true,
		},
		{
			name:     "add directive",
			out:      "ERROR: failed to solve: >>> ADD ./config /etc",
			expected: true,
		},
		{
			name:     "copy without marker",
			out:      "COPY ./bin /usr/bin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isFileNotFoundError(tt.out))
		})
	}
}

func TestFileNotFoundDetails(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("x"), 0o644))

	result := fileNotFoundDetails(dir)
	require.Contains(t, result, "not available in the build context")
	require.Contains(t, result, "main.go")
}

func TestFileNotFoundDetails_InvalidDir(t *testing.T) {
	result := fileNotFoundDetails(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Contains(t, result, "not available in the build context")
	require.NotContains(t, result, "\n")
}
