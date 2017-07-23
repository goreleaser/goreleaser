// Package testlib contains test helpers for goreleaser tests.
package testlib

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mktmp creates a new tempdir, cd into it and provides a back function that
// cd into the previous directory.
func Mktmp(t *testing.T) (folder string, back func()) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	current, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	return folder, func() {
		assert.NoError(os.Chdir(current))
	}
}
