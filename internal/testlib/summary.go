package testlib

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureSummary points the GitHub Actions job summary at a temporary file,
// and returns a function that reads back the lines written to it.
func CaptureSummary(tb testing.TB) func() []string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "summary.md")
	tb.Setenv("GITHUB_STEP_SUMMARY", path)
	return func() []string {
		bts, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		require.NoError(tb, err)
		return strings.Split(strings.TrimSuffix(string(bts), "\n"), "\n")
	}
}
