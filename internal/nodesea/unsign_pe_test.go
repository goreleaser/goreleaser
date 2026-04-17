package nodesea

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsignPE(t *testing.T) {
	t.Run("strips trailing certificate", func(t *testing.T) {
		b := &peBuilder{cert: bytes.Repeat([]byte{0xab}, 256)}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "signed.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		require.NoError(t, UnsignPE(path))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Len(t, got, len(raw)-256)
		va, size := peSecurityDir(got)
		require.Zero(t, va)
		require.Zero(t, size)

		stored, computed := peComputedChecksum(got)
		require.Equal(t, computed, stored, "stored checksum should match recomputed")
	})

	t.Run("unsigned is no-op", func(t *testing.T) {
		raw := (&peBuilder{}).build()
		path := filepath.Join(t.TempDir(), "unsigned.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))
		require.NoError(t, UnsignPE(path))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, raw, got)
	})

	t.Run("rejects non-tail certificate", func(t *testing.T) {
		b := &peBuilder{cert: []byte("CERT")}
		raw := b.build()
		raw = append(raw, 0xff)
		path := filepath.Join(t.TempDir(), "tailed.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))
		err := UnsignPE(path)
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
