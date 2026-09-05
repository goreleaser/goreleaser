package testlib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// CheckPath skips the test if the binary is not in PATH, unless CI=true.
// In CI, the test runs without checking this prerequisite.
func CheckPath(tb testing.TB, cmd string) {
	tb.Helper()
	if !InPath(cmd) {
		tb.Skipf("%s not in PATH", cmd)
	}
}

// IsCI returns true if we have the "CI" environment variable set to true.
func IsCI() bool {
	return os.Getenv("CI") == "true"
}

// InPath returns true if the given cmd is in the PATH, or if CI is true.
func InPath(cmd string) bool {
	if IsCI() {
		return true
	}
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsWindows returns true if current OS is Windows.
func IsWindows() bool { return runtime.GOOS == "windows" }

// SkipIfWindows skips the test if runtime OS is windows.
func SkipIfWindows(tb testing.TB, args ...any) {
	tb.Helper()
	if IsWindows() {
		tb.Skip(args...)
	}
}

// OnlyOnLinux skips the test unless the runtime OS is Linux.
func OnlyOnLinux(tb testing.TB, args ...any) {
	tb.Helper()
	if runtime.GOOS != "linux" {
		tb.Skip(args...)
	}
}

// Echo returns a `echo s` command, handling it on windows.
func Echo(s string) string {
	if IsWindows() {
		return "cmd.exe /c echo " + s
	}
	return "echo " + s
}

// Touch returns a `touch name` command, handling it on windows.
func Touch(name string) string {
	if IsWindows() {
		return "cmd.exe /c copy nul " + name
	}
	return "touch " + name
}

// ShC returns the command line for the given cmd wrapped into a `sh -c` in
// linux/mac, and the cmd.exe command in windows.
func ShC(cmd string) string {
	if IsWindows() {
		return fmt.Sprintf("cmd.exe /c '%s'", cmd)
	}
	return fmt.Sprintf("sh -c '%s'", cmd)
}

// Exit returns a command that exits the given status, handling windows.
func Exit(status int) string {
	if IsWindows() {
		return fmt.Sprintf("cmd.exe /c exit /b %d", status)
	}
	return fmt.Sprintf("exit %d", status)
}

// SharedZigCache points zig's global cache at a single directory shared by
// every test in the run, while the caller keeps a local cache of its own.
//
// The global cache is content addressed and lock protected, so sharing it is
// safe: 12 concurrent builds against one directory were verified to complete
// without error. Sharing it also lets each test reuse the libc and compiler-rt
// another test already compiled, which measured ~38% faster for the second and
// later projects.
//
// It deliberately lives outside the workspace. setup-zig caches
// $GITHUB_WORKSPACE/.zig-cache between runs, and restoring a half-saved copy of
// that is what actually produced the "manifest hit with missing outputs"
// corruption in goreleaser#6754, not concurrent access.
func SharedZigCache(tb testing.TB) {
	tb.Helper()
	// tb.TempDir() is deliberately not used: it is per-test, and the whole
	// point here is one directory shared by every test.
	dir := filepath.Join(os.TempDir(), "goreleaser-zig-global-cache") //nolint:usetesting
	if err := os.MkdirAll(dir, 0o755); err != nil {
		tb.Fatalf("could not create shared zig cache: %v", err)
	}
	tb.Setenv("ZIG_GLOBAL_CACHE_DIR", dir)
}
