// Package testlib contains test helpers for goreleaser tests.
package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Mktmp creates a new tempdir, cd into it and automatically cd back when the
// test finishes.
func Mktmp(tb testing.TB) string {
	tb.Helper()
	folder := tb.TempDir()
	current, err := os.Getwd()
	require.NoError(tb, err)
	require.NoError(tb, os.Chdir(folder))
	tb.Cleanup(func() {
		require.NoError(tb, os.Chdir(current))
	})
	return folder
}
