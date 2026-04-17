package nodesea

import (
	"bytes"
	"debug/macho"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectMachO(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		text := make([]byte, 256)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{
			textData:     text,
			linkeditData: bytes.Repeat([]byte{0xaa}, 32),
			slack:        4096, // room for new LC
		}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "host")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		blob := []byte("a sea blob payload")
		require.NoError(t, InjectMachO(path, blob))

		got, err := os.ReadFile(path)
		require.NoError(t, err)

		f, err := macho.NewFile(newReadSeeker(got))
		require.NoError(t, err)
		seg := f.Segment(MachOSegmentName)
		require.NotNil(t, seg, "segment must exist")
		require.Equal(t, uint64(len(blob)), seg.Filesz)

		require.Contains(t, string(got), SentinelFused)
	})

	t.Run("rejects double injection", func(t *testing.T) {
		text := make([]byte, 256)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{textData: text, linkeditData: []byte("LE"), slack: 4096}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "host")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		require.NoError(t, InjectMachO(path, []byte("blob")))
		err := InjectMachO(path, []byte("blob"))
		require.ErrorIs(t, err, ErrAlreadyInjected)
	})

	t.Run("rejects insufficient slack", func(t *testing.T) {
		text := make([]byte, 64)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{textData: text, linkeditData: []byte("LE"), slack: 8}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "host")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		err := InjectMachO(path, []byte("blob"))
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
