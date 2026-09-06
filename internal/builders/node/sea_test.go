package node

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestSignMachO(t *testing.T) {
	testlib.CheckPath(t, "go")
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	path := filepath.Join(dir, "macho")
	require.NoError(t, os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o644))
	cmd := exec.CommandContext(t.Context(), "go", "build", "-ldflags=-s -w", "-o", path, source)
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=arm64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	// Node leaves a signature command pointing past EOF. Keep that case as
	// well as replacing the valid signature produced by the Go linker.
	for name, contents := range map[string][]byte{
		"signed":      original,
		"placeholder": original[:machoSignatureOffset(t, original)],
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "macho")
			require.NoError(t, os.WriteFile(path, contents, 0o755))
			const identity = "goreleaser-node-sign-test"
			require.NoError(t, signMachO(path, identity))

			signed, err := os.ReadFile(path)
			require.NoError(t, err)
			signature := signed[machoSignatureOffset(t, signed):]
			require.GreaterOrEqual(t, len(signature), 8)
			// Embedded Mach-O signatures use a big-endian superblob header.
			require.Equal(t, uint32(0xfade0cc0), binary.BigEndian.Uint32(signature[:4]))
			size := binary.BigEndian.Uint32(signature[4:8])
			require.LessOrEqual(t, uint64(size), uint64(len(signature)))
			require.Contains(t, string(signature[:size]), identity+"\x00")
		})
	}
}

func machoSignatureOffset(tb testing.TB, contents []byte) uint32 {
	tb.Helper()
	file, err := macho.NewFile(bytes.NewReader(contents))
	require.NoError(tb, err)
	const codeSignatureCommand = 0x1d
	for _, load := range file.Loads {
		raw := load.Raw()
		if file.ByteOrder.Uint32(raw[:4]) != codeSignatureCommand {
			continue
		}
		require.Len(tb, raw, 16)
		offset := file.ByteOrder.Uint32(raw[8:12])
		require.LessOrEqual(tb, uint64(offset), uint64(len(contents)))
		return offset
	}
	tb.Fatal("missing Mach-O code signature command")
	return 0
}
