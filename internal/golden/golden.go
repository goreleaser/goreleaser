// Package golden asserts golden files contents.
package golden

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

const golden = ".golden"

// RequireEqual requires the output to be equal to the golden file.
func RequireEqual(tb testing.TB, out []byte) {
	tb.Helper()
	RequireEqualExt(tb, out, "")
}

// RequireEqualExt requires the output to be equal to the golden file with the
// given extension.
func RequireEqualExt(tb testing.TB, out []byte, ext string) {
	tb.Helper()
	doRequireEqual(tb, out, ext, golden, false)
}

// RequireEqualExtSubfolder requires the output to be equal to the golden file.
func RequireEqualExtSubfolder(tb testing.TB, out []byte, ext string) {
	tb.Helper()
	doRequireEqual(tb, out, ext, golden, true)
}

// RequireReadFile requires the file to be read and returned.
func RequireReadFile(tb testing.TB, path string) []byte {
	tb.Helper()
	bts, err := os.ReadFile(path)
	require.NoError(tb, err)
	return bts
}

func doRequireEqual(tb testing.TB, out []byte, ext, suffix string, folder bool) {
	tb.Helper()

	out = fixLineEndings(out)

	golden := filepath.Join("testdata", tb.Name()+ext+suffix)
	if folder {
		golden = filepath.Join("testdata", tb.Name(), filepath.Base(tb.Name())+ext+suffix)
	}
	if *update {
		require.NoError(tb, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(tb, os.WriteFile(golden, out, 0o655))
	}

	gbts, err := os.ReadFile(golden)
	require.NoError(tb, err)
	gbts = fixLineEndings(gbts)

	require.Equal(tb, string(gbts), string(out))
}

func fixLineEndings(in []byte) []byte {
	return bytes.ReplaceAll(in, []byte("\r\n"), []byte{'\n'})
}
