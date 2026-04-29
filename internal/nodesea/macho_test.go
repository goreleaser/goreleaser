package nodesea

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsignMachOBytes(t *testing.T) {
	t.Run("strips trailing signature and updates linkedit vmsize", func(t *testing.T) {
		b := &machoBuilder{
			textData:       []byte("TEXT_PAYLOAD"),
			linkeditData:   bytes.Repeat([]byte{0xaa}, 64),
			signatureBytes: bytes.Repeat([]byte{0xcc}, 128),
		}
		raw := b.build()

		got, err := unsignMachOBytes(raw)
		require.NoError(t, err)
		require.Len(t, got, len(raw)-128, "file should be truncated by signature size")

		f, err := macho.NewFile(bytes.NewReader(got))
		require.NoError(t, err)
		require.Equal(t, uint32(2), f.Ncmd, "LC count should drop by 1")
		linkedit := f.Segment("__LINKEDIT")
		require.NotNil(t, linkedit)
		require.Equal(t, uint64(64), linkedit.Filesz, "linkedit shrinks back to original size")
		// Regression test for the previously-missing vmsize update:
		// codesign rejects a __LINKEDIT whose vmsize > filesize.
		require.Equal(t, uint64(64), linkedit.Memsz, "linkedit vmsize must match filesize")

		require.NotContains(t, string(got[:32+f.Cmdsz]), string([]byte{0x1d, 0, 0, 0}))
	})

	t.Run("no signature is no-op", func(t *testing.T) {
		b := &machoBuilder{
			textData:     []byte("TEXT"),
			linkeditData: []byte("LINKEDIT"),
		}
		raw := b.build()
		got, err := unsignMachOBytes(raw)
		require.NoError(t, err)
		require.Equal(t, raw, got)
	})

	t.Run("rejects fat", func(t *testing.T) {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint32(buf[:4], macho.MagicFat)
		_, err := unsignMachOBytes(buf)
		require.ErrorIs(t, err, ErrNotSupported)
	})

	t.Run("rejects 32-bit", func(t *testing.T) {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint32(buf[:4], macho.Magic32)
		_, err := unsignMachOBytes(buf)
		require.ErrorIs(t, err, ErrNotSupported)
	})

	t.Run("rejects non-tail signature", func(t *testing.T) {
		b := &machoBuilder{
			textData:       []byte("TEXT"),
			linkeditData:   []byte("LE"),
			signatureBytes: []byte("SIG"),
		}
		raw := b.build()
		raw = append(raw, 0xff, 0xff)
		_, err := unsignMachOBytes(raw)
		require.ErrorIs(t, err, ErrNotSupported)
	})
}

func TestInjectMachOBytes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		text := make([]byte, 256)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{
			textData:     text,
			linkeditData: bytes.Repeat([]byte{0xaa}, 32),
			slack:        4096,
		}
		raw := b.build()

		blob := []byte("a sea blob payload")
		got, err := injectMachOBytes(raw, blob)
		require.NoError(t, err)

		f, err := macho.NewFile(bytes.NewReader(got))
		require.NoError(t, err)
		seg := f.Segment(MachOSegmentName)
		require.NotNil(t, seg, "segment must exist")
		// Segment filesize is page-aligned; the embedded section is
		// the one whose size matches the blob.
		require.GreaterOrEqual(t, seg.Filesz, uint64(len(blob)))
		sect := f.Section(MachOSectionName)
		require.NotNil(t, sect, "section must exist")
		require.Equal(t, uint64(len(blob)), sect.Size)
	})

	t.Run("rejects double injection", func(t *testing.T) {
		text := make([]byte, 256)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{textData: text, linkeditData: []byte("LE"), slack: 4096}
		raw := b.build()

		out, err := injectMachOBytes(raw, []byte("blob"))
		require.NoError(t, err)
		_, err = injectMachOBytes(out, []byte("blob"))
		require.ErrorIs(t, err, ErrAlreadyInjected)
	})

	t.Run("rejects insufficient slack", func(t *testing.T) {
		text := make([]byte, 64)
		copy(text[16:], []byte(SentinelStock))
		b := &machoBuilder{textData: text, linkeditData: []byte("LE"), slack: 8}
		raw := b.build()

		_, err := injectMachOBytes(raw, []byte("blob"))
		require.ErrorIs(t, err, ErrNotSupported)
	})
}

func TestFlipMachOSentinel(t *testing.T) {
	t.Run("flips :0 to :1", func(t *testing.T) {
		data := []byte("padding " + SentinelStock + " more")
		got, err := flipSentinelBytes(data)
		require.NoError(t, err)
		require.Contains(t, string(got), SentinelFused)
	})

	t.Run("rejects already fused", func(t *testing.T) {
		data := []byte("padding " + SentinelFused + " more")
		_, err := flipSentinelBytes(data)
		require.ErrorIs(t, err, ErrAlreadyFused)
	})

	t.Run("rejects missing sentinel", func(t *testing.T) {
		data := []byte("nothing to see here")
		_, err := flipSentinelBytes(data)
		require.ErrorIs(t, err, ErrSentinelNotFound)
	})
}
