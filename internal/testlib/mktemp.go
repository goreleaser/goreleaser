// Package testlib contains test helpers for goreleaser tests.
package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Mktmp creates a new tempdir, cd into it and provides a back function that
// cd into the previous directory.
func Mktmp(t testing.TB) string {
	var folder = t.TempDir()
	current, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(folder))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(current))
	})
	return folder
}
