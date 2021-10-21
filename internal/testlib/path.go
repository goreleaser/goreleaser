package testlib

import (
	"os"
	"os/exec"
	"testing"
)

// CheckPath skips the test if the binary is not in the PATH.
func CheckPath(tb testing.TB, cmd string) {
	tb.Helper()
	if os.Getenv("CI") == "true" {
		// never skip on CI
		return
	}
	if _, err := exec.LookPath(cmd); err != nil {
		tb.Skipf("%s not in PATH", cmd)
	}
}
