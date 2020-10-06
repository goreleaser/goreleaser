// Package testlib contains test helpers for goreleaser tests.
package testlib

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Mktmp creates a new tempdir, cd into it and provides a back function that
// cd into the previous directory.
func Mktmp(t *testing.T) (folder string, back func()) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	require.NoError(t, err)
	current, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(folder))
	return folder, func() {
		require.NoError(t, os.Chdir(current))
	}
}
