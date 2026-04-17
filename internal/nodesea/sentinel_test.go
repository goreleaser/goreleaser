package nodesea

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlipSentinel(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fake-node")
		body := append([]byte("prefix\x00\x00"), []byte(SentinelStock)...)
		body = append(body, 0xff, 0xee)
		require.NoError(t, os.WriteFile(path, body, 0o644))

		require.NoError(t, FlipSentinel(path))

		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(got), SentinelFused)
		require.NotContains(t, string(got), SentinelStock)
		// trailing bytes preserved
		idx := bytes.Index(got, []byte(SentinelFused))
		require.Equal(t, byte(0xff), got[idx+len(SentinelFused)])
	})

	t.Run("missing sentinel", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "not-node")
		require.NoError(t, os.WriteFile(path, []byte("hello world"), 0o644))
		err := FlipSentinel(path)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrSentinelNotFound))
	})

	t.Run("already fused", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fused")
		body := append([]byte("prefix"), []byte(SentinelFused)...)
		require.NoError(t, os.WriteFile(path, body, 0o644))
		err := FlipSentinel(path)
		require.ErrorIs(t, err, ErrAlreadyFused)
	})

	t.Run("ambiguous", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "dup")
		body := append([]byte(SentinelStock), []byte(SentinelStock)...)
		require.NoError(t, os.WriteFile(path, body, 0o644))
		err := FlipSentinel(path)
		require.ErrorIs(t, err, ErrSentinelAmbiguous)
	})

	t.Run("malformed marker", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "malformed")
		body := append([]byte(Sentinel), 'x', '0')
		require.NoError(t, os.WriteFile(path, body, 0o644))
		err := FlipSentinel(path)
		require.ErrorIs(t, err, ErrSentinelNotFound)
	})
}

func TestFormatFor(t *testing.T) {
	require.Equal(t, FormatELF, FormatFor("linux"))
	require.Equal(t, FormatMachO, FormatFor("darwin"))
	require.Equal(t, FormatPE, FormatFor("windows"))
	require.Equal(t, Format(0), FormatFor("plan9"))
	require.Equal(t, "elf", FormatELF.String())
	require.Equal(t, "macho", FormatMachO.String())
	require.Equal(t, "pe", FormatPE.String())
	require.Equal(t, "unknown", Format(0).String())
}
