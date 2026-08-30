// Package testlib contains test helpers for goreleaser tests.
package testlib

import (
	"testing"
)

// Mktmp creates a new tempdir, cd into it and automatically cd back when the
// test finishes.
func Mktmp(tb testing.TB) string {
	tb.Helper()
	folder := tb.TempDir()
	tb.Chdir(folder)
	return folder
}
