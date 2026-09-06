package testlib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedZigCache(t *testing.T) {
	for name, runnerTemp := range map[string]string{
		"runner":   t.TempDir(),
		"fallback": "",
	} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("TMP", tmp)
			t.Setenv("TEMP", tmp)
			t.Setenv("RUNNER_TEMP", runnerTemp)
			if runnerTemp != "" {
				tmp = runnerTemp
			}

			SharedZigCache(t)

			dir := filepath.Join(tmp, "goreleaser-zig-global-cache")
			require.Equal(t, dir, os.Getenv("ZIG_GLOBAL_CACHE_DIR"))
			require.DirExists(t, dir)
		})
	}
}

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
