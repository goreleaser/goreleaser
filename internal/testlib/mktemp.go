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

// ScopeTempDir points os.TempDir() at a fresh per-test directory and
// returns it. Use it to assert on what the code under test leaves
// behind, and so that whatever it does leave dies with the test.
func ScopeTempDir(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	// os.TempDir consults these on every call: TMPDIR on unix, TMP and
	// TEMP on windows.
	tb.Setenv("TMPDIR", dir)
	tb.Setenv("TMP", dir)
	tb.Setenv("TEMP", dir)
	return dir
}
