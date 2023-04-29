package client

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGitClient(t *testing.T) {
	keypath := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")
	url := testlib.GitMakeBareRpository(t)
	ctx := testctx.NewWithCfg(config.Project{
		Dist: t.TempDir(),
	})

	cli, err := NewGitUploadClient(ctx)
	require.NoError(t, err)

	require.NoError(t, cli.CreateFile(
		ctx,
		config.CommitAuthor{
			Name:  "Foo",
			Email: "foo@bar.com",
		},
		Repo{
			GitURL:        url,
			GitSSHCommand: DefaulGittSSHCommand,
			PrivateKey:    keypath,
			Name:          "test1",
		},
		[]byte("fake content"),
		"fake.txt",
		"hey test",
	))
}

func TestKeyPath(t *testing.T) {
	t.Run("with valid path", func(t *testing.T) {
		path := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")
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
		path := testlib.MakeNewSSHKey(t, keygen.Ed25519, "pwd")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(string(bts))
		require.EqualError(t, err, "key is password-protected")
		require.Empty(t, result)
	})

	t.Run("with key", func(t *testing.T) {
		for _, algo := range []keygen.KeyType{keygen.Ed25519, keygen.RSA} {
			t.Run(string(algo), func(t *testing.T) {
				path := testlib.MakeNewSSHKey(t, algo, "")
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
		path := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(strings.TrimSpace(string(bts)))
		require.NoError(t, err)

		resultbts, err := os.ReadFile(result)
		require.NoError(t, err)
		require.Equal(t, string(bts), string(resultbts))
	})
}
