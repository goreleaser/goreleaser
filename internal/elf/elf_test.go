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

	t.Run("statically linked", func(t *testing.T) {
		t.Parallel()
		tmp := createTempFile(t, staticallyLinked)

		binPath := filepath.Join(tmp, "bin")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "go build failed: %s", output)

		require.False(t, IsDynamicallyLinked(binPath))
	})

	t.Run("dynamically linked", func(t *testing.T) {
		t.Parallel()
		tmp := createTempFile(t, dynamicallyLinked)

		binPath := filepath.Join(tmp, "bin")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "go build failed: %s", output)

		require.True(t, IsDynamicallyLinked(binPath), "binary with cgo_import_dynamic should be dynamically linked")
	})
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
