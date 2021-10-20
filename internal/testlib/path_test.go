package testlib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckPath(t *testing.T) {
	requireSkipped := func(tb testing.TB, skipped bool) {
		tb.Helper()
		t.Cleanup(func() {
			require.Equalf(tb, skipped, tb.Skipped(), "expected skipped to be %v", skipped)
		})
	}

	t.Run("local", func(t *testing.T) {
		t.Run("in path", func(t *testing.T) {
			requireSkipped(t, false)
			CheckPath(t, "echo")
		})

		t.Run("not in path", func(t *testing.T) {
			requireSkipped(t, true)
			CheckPath(t, "do-not-exist")
		})
	})

	t.Run("CI", func(t *testing.T) {
		require.NoError(t, os.Setenv("CI", "true"))
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv("CI"))
		})

		t.Run("in path on CI", func(t *testing.T) {
			requireSkipped(t, false)
			CheckPath(t, "echo")
		})

		t.Run("not in path on CI", func(t *testing.T) {
			requireSkipped(t, false)
			CheckPath(t, "do-not-exist")
		})
	})
}
