package elf

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsDynamicallyLinked(t *testing.T) {
	t.Parallel()

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()
		require.False(t, IsDynamicallyLinked("/does/not/exist"))
	})

	for _, tc := range []struct {
		name   string
		source string
		want   bool
	}{
		{"statically linked", staticallyLinked, false},
		{"dynamically linked", dynamicallyLinked, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmp := createTempFile(t, tc.source)

			binPath := filepath.Join(tmp, "bin")
			cmd := exec.Command("go", "build", "-o", binPath, ".")
			cmd.Dir = tmp
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "go build failed: %s", output)

			require.Equal(t, tc.want, IsDynamicallyLinked(binPath))
			require.Equal(t, !tc.want, IsStaticallyLinked(binPath))
			if tc.want {
				// The test host may be glibc or musl; both are valid dynamic
				// Linux runtimes and the detector must recognize either.
				require.Contains(t, []string{"gnu", "musl"}, Libc(binPath))
			} else {
				require.Empty(t, Libc(binPath))
			}
		})
	}
}

// Not being dynamic does not make a file static: it must be an ELF executable.
func TestIsStaticallyLinkedNeedsELF(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "script")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755))
	require.False(t, IsDynamicallyLinked(script))
	require.False(t, IsStaticallyLinked(script))
	require.False(t, IsStaticallyLinked(filepath.Join(dir, "missing")))
}

func TestLibcFromInterpreter(t *testing.T) {
	for interpreter, want := range map[string]string{
		"/lib64/ld-linux-x86-64.so.2": "gnu",
		"/lib64/ld64.so.2":            "gnu", // ppc64le
		"/lib64/ld64.so.1":            "gnu", // ppc64
		"/lib/ld.so.1":                "gnu", // mips
		"/lib/ld64.so.1":              "gnu", // s390x
		"/lib/ld-musl-aarch64.so.1":   "musl",
		"/lib/ld-uClibc.so.0":         "",
		"":                            "",
	} {
		require.Equal(t, want, libcFromInterpreter(interpreter), interpreter)
	}
}

func createTempFile(tb testing.TB, maingo string) string {
	tb.Helper()
	tmp := tb.TempDir()
	require.NoError(tb, os.WriteFile(filepath.Join(tmp, "main.go"), []byte(maingo), 0o644))
	require.NoError(tb, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644))
	return tmp
}

const goMod = `module test
go 1.21
`

const staticallyLinked = `package main

func main() {}
`

const dynamicallyLinked = `package main

import _ "unsafe"

// Import a libc function to force dynamic linking
//go:cgo_import_dynamic libc_puts puts "libc.so.6"

func main() {}
`
