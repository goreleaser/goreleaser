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

	setupEnv := func(tb testing.TB, value string) {
		tb.Helper()
		previous := os.Getenv("CI")
		require.NoError(tb, os.Setenv("CI", value))
		tb.Cleanup(func() {
			require.NoError(tb, os.Setenv("CI", previous))
		})
	}

	t.Run("local", func(t *testing.T) {
		setupEnv(t, "false")

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
		setupEnv(t, "true")

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
