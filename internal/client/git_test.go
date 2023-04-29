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
	cli := NewGitUploadClient("master")
	author := config.CommitAuthor{
		Name:  "Foo",
		Email: "foo@bar.com",
	}

	t.Run("full", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     url,
			PrivateKey: testlib.MakeNewSSHKey(t, keygen.Ed25519, ""),
			Name:       "test1",
		}
		require.NoError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("fake content"),
			"fake.txt",
			"hey test",
		))
		require.NoError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("fake content 2"),
			"fake.txt",
			"hey test 2",
		))
		require.Equal(t, "fake content 2", string(testlib.CatFileFromBareRepository(t, url, "fake.txt")))
	})
	t.Run("no repo name", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     url,
			PrivateKey: testlib.MakeNewSSHKey(t, keygen.Ed25519, ""),
		}
		require.NoError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("fake content"),
			"fake.txt",
			"hey test",
		))
		require.NoError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("fake content 2"),
			"fake.txt",
			"hey test 2",
		))
		require.Equal(t, "fake content 2", string(testlib.CatFileFromBareRepository(t, url, "fake.txt")))
	})
	t.Run("bad url", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL: "{{ .Nope }}",
		}
		testlib.RequireTemplateError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		))
	})
	t.Run("bad ssh cmd", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:        testlib.GitMakeBareRepository(t),
			PrivateKey:    testlib.MakeNewSSHKey(t, keygen.Ed25519, ""),
			GitSSHCommand: "{{.Foo}}",
		}
		testlib.RequireTemplateError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		))
	})
	t.Run("empty url", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{}
		require.EqualError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		), "url is empty")
	})
	t.Run("bad ssh cmd", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     testlib.GitMakeBareRepository(t),
			PrivateKey: "{{.Foo}}",
		}
		testlib.RequireTemplateError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		))
	})
	t.Run("bad key path", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     testlib.GitMakeBareRepository(t),
			PrivateKey: "./nope",
		}
		require.Error(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		))
	})
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
		require.EqualError(t, err, "git: key is password-protected")
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
		require.EqualError(t, err, `private_key is empty`)
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
