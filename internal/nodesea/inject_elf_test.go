package nodesea

import (
	"debug/elf"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectELF(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		raw := elfBuilder{}.build()
		path := filepath.Join(t.TempDir(), "host")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		blob := []byte("hello sea blob")
		require.NoError(t, InjectELF(path, blob))

		got, err := os.ReadFile(path)
		require.NoError(t, err)

		// Reparse and verify a PT_NOTE phdr exists pointing at a SHT_NOTE
		// section whose note matches our name+type+desc.
		f, err := elf.NewFile(newReadSeeker(got))
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
		path := filepath.Join(t.TempDir(), "host")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		require.NoError(t, InjectELF(path, []byte("blob")))
		err := InjectELF(path, []byte("blob"))
		require.ErrorIs(t, err, ErrAlreadyInjected)
	})

	t.Run("rejects non-elf", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "notelf")
		require.NoError(t, os.WriteFile(path, []byte("not an ELF file"), 0o755))
		err := InjectELF(path, []byte("blob"))
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
