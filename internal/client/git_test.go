package client

import (
	"os"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGitClient(t *testing.T) {
	t.Parallel()

	sshKey := testlib.MakeNewSSHKey(t, "")

	author := config.CommitAuthor{
		Name:  "Foo",
		Email: "foo@bar.com",
	}

	t.Run("bad url", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL: "{{ .Nope }}",
		}
		cli := NewGitUploadClient(repo.Branch)
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:        testlib.GitMakeBareRepository(t),
			PrivateKey:    sshKey,
			GitSSHCommand: "{{.Foo}}",
		}
		cli := NewGitUploadClient(repo.Branch)
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
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{}
		cli := NewGitUploadClient(repo.Branch)
		require.EqualError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		), "url is empty")
	})
	t.Run("bad ssh cmd template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     testlib.GitMakeBareRepository(t),
			PrivateKey: "{{.Foo}}",
		}
		cli := NewGitUploadClient(repo.Branch)
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     testlib.GitMakeBareRepository(t),
			PrivateKey: "./nope",
		}
		cli := NewGitUploadClient(repo.Branch)
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
	t.Parallel()

	sshKey := testlib.MakeNewSSHKey(t, "")

	t.Run("with valid path", func(t *testing.T) {
		t.Parallel()
		result, err := keyPath(sshKey)
		require.NoError(t, err)
		require.Equal(t, sshKey, result)
	})
	t.Run("with invalid path", func(t *testing.T) {
		t.Parallel()
		result, err := keyPath("testdata/nope")
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Empty(t, result)
	})

	t.Run("with password protected key path", func(t *testing.T) {
		t.Parallel()
		path := testlib.MakeNewSSHKey(t, "pwd")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(string(bts))
		require.EqualError(t, err, "git: key is password-protected")
		require.Empty(t, result)
	})

	t.Run("with key", func(t *testing.T) {
		t.Parallel()

		_, err := keyPath(sshKey)
		require.NoError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		result, err := keyPath("")
		require.EqualError(t, err, `private_key is empty`)
		require.Empty(t, result)
	})

	t.Run("with invalid EOF", func(t *testing.T) {
		t.Parallel()
		bts, err := os.ReadFile(sshKey)
		require.NoError(t, err)

		result, err := keyPath(strings.TrimSpace(string(bts)))
		require.NoError(t, err)

		resultbts, err := os.ReadFile(result)
		require.NoError(t, err)
		require.Equal(t, string(bts), string(resultbts))
	})
}

func TestRepoFromURL(t *testing.T) {
	t.Parallel()
	for k, v := range map[string]string{
		"goreleaser": "git@github.com:goreleaser/goreleaser.git",
		"nfpm":       "https://github.com/goreleaser/nfpm",
		"test":       "https://myserver.git/foo/test.git",
	} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, k, nameFromURL(v))
		})
	}
}
