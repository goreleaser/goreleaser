package nodesea

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsignMachO(t *testing.T) {
	t.Run("strips trailing signature", func(t *testing.T) {
		b := &machoBuilder{
			textData:       []byte("TEXT_PAYLOAD"),
			linkeditData:   bytes.Repeat([]byte{0xaa}, 64),
			signatureBytes: bytes.Repeat([]byte{0xcc}, 128),
		}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "thin")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		require.NoError(t, UnsignMachO(path))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Len(t, got, len(raw)-128, "file should be truncated by signature size")

		// Re-parse and check structure.
		f, err := macho.NewFile(newReadSeeker(got))
		require.NoError(t, err)
		require.Equal(t, uint32(2), f.Ncmd, "LC count should drop by 1")
		linkedit := f.Segment("__LINKEDIT")
		require.NotNil(t, linkedit)
		require.Equal(t, uint64(64), linkedit.Filesz, "linkedit shrinks back to original size")

		// Sentinel bytes for LC_CODE_SIGNATURE should be gone.
		// Old LC area still has zeros at the tail; that's fine.
		require.NotContains(t, string(got[:32+f.Cmdsz]), string([]byte{0x1d, 0, 0, 0}))
	})

	t.Run("no signature is no-op", func(t *testing.T) {
		b := &machoBuilder{
			textData:     []byte("TEXT"),
			linkeditData: []byte("LINKEDIT"),
		}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "unsigned")
		require.NoError(t, os.WriteFile(path, raw, 0o755))
		require.NoError(t, UnsignMachO(path))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, raw, got)
	})

	t.Run("rejects fat", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fat")
		buf := make([]byte, 8)
		binary.BigEndian.PutUint32(buf[:4], macho.MagicFat)
		require.NoError(t, os.WriteFile(path, buf, 0o755))
		err := UnsignMachO(path)
		require.ErrorIs(t, err, ErrNotSupported)
	})

	t.Run("rejects 32-bit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "thin32")
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint32(buf[:4], macho.Magic32)
		require.NoError(t, os.WriteFile(path, buf, 0o755))
		err := UnsignMachO(path)
		require.ErrorIs(t, err, ErrNotSupported)
	})

	t.Run("rejects non-tail signature", func(t *testing.T) {
		// Build a normal signed binary then pad bytes after it so the
		// signature is no longer at EOF.
		b := &machoBuilder{
			textData:       []byte("TEXT"),
			linkeditData:   []byte("LE"),
			signatureBytes: []byte("SIG"),
		}
		raw := b.build()
		raw = append(raw, 0xff, 0xff)
		path := filepath.Join(t.TempDir(), "notail")
		require.NoError(t, os.WriteFile(path, raw, 0o755))
		err := UnsignMachO(path)
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
