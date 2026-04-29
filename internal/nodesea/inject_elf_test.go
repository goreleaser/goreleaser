package nodesea

import (
	"bytes"
	"debug/elf"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectELF(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		raw := elfBuilder{}.build()
		blob := []byte("hello sea blob")

		got, err := injectELFBytes(raw, blob)
		require.NoError(t, err)

		// Reparse and verify a PT_NOTE phdr exists pointing at a SHT_NOTE
		// section whose note matches our name+type+desc.
		f, err := elf.NewFile(bytes.NewReader(got))
		require.NoError(t, err)

		require.True(t, findElfNote(f, got, noteName, noteType), "note must be present after injection")

		var sawPhdr bool
		for _, p := range f.Progs {
			if p.Type == elf.PT_NOTE {
				sawPhdr = true
				break
			}
		}
		require.True(t, sawPhdr, "PT_NOTE phdr must be present")

		// Sentinel must be flipped.
		require.Contains(t, string(got), SentinelFused)
	})

	t.Run("rejects double injection", func(t *testing.T) {
		raw := elfBuilder{}.build()

		out, err := injectELFBytes(raw, []byte("blob"))
		require.NoError(t, err)
		_, err = injectELFBytes(out, []byte("blob"))
		require.ErrorIs(t, err, ErrAlreadyInjected)
	})

	t.Run("rejects non-elf", func(t *testing.T) {
		_, err := injectELFBytes([]byte("not an ELF file"), []byte("blob"))
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
