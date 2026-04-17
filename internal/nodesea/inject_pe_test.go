package nodesea

import (
	"bytes"
	"debug/pe"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInjectPE(t *testing.T) {
	t.Run("happy path with sentinel in .text", func(t *testing.T) {
		text := make([]byte, 0x200)
		copy(text[16:], []byte(SentinelStock))
		b := &peBuilder{rsrcSize: 0x200, textData: text}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "host.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		blob := []byte("a sea blob payload!")
		require.NoError(t, InjectPE(path, blob))

		got, err := os.ReadFile(path)
		require.NoError(t, err)

		f, err := pe.NewFile(newReadSeeker(got))
		require.NoError(t, err)

		var rsrc *pe.Section
		for _, s := range f.Sections {
			if s.Name == ".rsrc" {
				rsrc = s
			}
		}
		require.NotNil(t, rsrc)

		// Parse the new resource tree and locate our entry.
		raw2, err := rsrc.Data()
		require.NoError(t, err)
		tree, err := parseResourceDir(raw2, 0, rsrc.VirtualAddress)
		require.NoError(t, err)
		entry := tree.find(rtRCData, PEResourceName)
		require.NotNil(t, entry, "NODE_SEA_BLOB resource missing")
		require.NotEmpty(t, entry.dir.entries)
		leaf := entry.dir.entries[0]
		require.Equal(t, blob, leaf.data)

		// Sentinel flipped.
		require.True(t, bytes.Contains(got, []byte(SentinelFused)))
	})

	t.Run("rejects double injection", func(t *testing.T) {
		text := make([]byte, 0x200)
		copy(text[16:], []byte(SentinelStock))
		b := &peBuilder{rsrcSize: 0x200, textData: text}
		raw := b.build()
		path := filepath.Join(t.TempDir(), "host.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))

		require.NoError(t, InjectPE(path, []byte("blob")))
		err := InjectPE(path, []byte("blob"))
		require.ErrorIs(t, err, ErrAlreadyInjected)
	})

	t.Run("rejects no rsrc", func(t *testing.T) {
		raw := (&peBuilder{}).build()
		path := filepath.Join(t.TempDir(), "host.exe")
		require.NoError(t, os.WriteFile(path, raw, 0o755))
		err := InjectPE(path, []byte("blob"))
		require.ErrorIs(t, err, ErrNotSupported)
	})
}
