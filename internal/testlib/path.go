package testlib

import (
	"os"
	"os/exec"
	"testing"
)

// CheckPath skips the test if the binary is not in the PATH, or if CI is true.
func CheckPath(tb testing.TB, cmd string) {
	tb.Helper()
	if !InPath(cmd) {
		tb.Skipf("%s not in PATH", cmd)
	}
}

// InPath returns true if the given cmd is in the PATH, or if CI is true.
func InPath(cmd string) bool {
	if os.Getenv("CI") == "true" {
		return true
	}
	_, err := exec.LookPath(cmd)
	return err == nil
}
