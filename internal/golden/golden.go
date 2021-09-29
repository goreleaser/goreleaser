// Package golden asserts golden files contents.
package golden

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update .golden files")

func RequireEqual(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, "")
}

func RequireEqualTxt(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, ".txt")
}

func RequireEqualJSON(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, ".json")
}

func RequireEqualRb(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, ".rb")
}

func RequireEqualLua(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, ".lua")
}

func RequireEqualYaml(tb testing.TB, out []byte) {
	tb.Helper()
	doRequireEqual(tb, out, ".yml")
}

func doRequireEqual(tb testing.TB, out []byte, ext string) {
	tb.Helper()

	golden := "testdata/" + tb.Name() + ext + ".golden"
	if *update {
		require.NoError(tb, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(tb, os.WriteFile(golden, out, 0o655))
	}

	gbts, err := os.ReadFile(golden)
	require.NoError(tb, err)

	require.Equal(tb, string(gbts), string(out))
}
