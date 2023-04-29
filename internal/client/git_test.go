package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/stretchr/testify/require"
)

func TestKeyPath(t *testing.T) {
	t.Run("with valid path", func(t *testing.T) {
		path := makeKey(t, keygen.Ed25519, "")
		result, err := keyPath(path)
		require.NoError(t, err)
		require.Equal(t, path, result)
	})
	t.Run("with invalid path", func(t *testing.T) {
		result, err := keyPath("testdata/nope")
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Equal(t, "", result)
	})

	t.Run("with password protected key path", func(t *testing.T) {
		path := makeKey(t, keygen.Ed25519, "pwd")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(string(bts))
		require.EqualError(t, err, "key is password-protected")
		require.Empty(t, result)
	})

	t.Run("with key", func(t *testing.T) {
		for _, algo := range []keygen.KeyType{keygen.Ed25519, keygen.RSA} {
			t.Run(string(algo), func(t *testing.T) {
				path := makeKey(t, algo, "")
				bts, err := os.ReadFile(path)
				require.NoError(t, err)

				result, err := keyPath(string(bts))
				require.NoError(t, err)

				resultbts, err := os.ReadFile(result)
				require.NoError(t, err)
				require.Equal(t, string(bts), string(resultbts))
			})
		}
	})
	t.Run("empty", func(t *testing.T) {
		result, err := keyPath("")
		require.EqualError(t, err, `aur.private_key is empty`)
		require.Equal(t, "", result)
	})
	t.Run("with invalid EOF", func(t *testing.T) {
		path := makeKey(t, keygen.Ed25519, "")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(strings.TrimSpace(string(bts)))
		require.NoError(t, err)

		resultbts, err := os.ReadFile(result)
		require.NoError(t, err)
		require.Equal(t, string(bts), string(resultbts))
	})
}

func makeKey(tb testing.TB, algo keygen.KeyType, pass string) string {
	tb.Helper()

	dir := tb.TempDir()
	filepath := filepath.Join(dir, "id_"+algo.String())
	_, err := keygen.New(filepath, keygen.WithKeyType(algo), keygen.WithWrite(), keygen.WithPassphrase(pass))
	require.NoError(tb, err)
	return filepath
}
