package testlib

import (
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
		t.Setenv("CI", "false")

		t.Run("in path", func(t *testing.T) {
			requireSkipped(t, false)
			if IsWindows() {
				CheckPath(t, "cmd.exe")
			} else {
				CheckPath(t, "echo")
			}
		})

		t.Run("not in path", func(t *testing.T) {
			requireSkipped(t, true)
			CheckPath(t, "do-not-exist")
		})
	})

	t.Run("CI", func(t *testing.T) {
		t.Setenv("CI", "true")

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
