package client

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGitClient(t *testing.T) {
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
			PrivateKey: testlib.MakeNewSSHKey(t, ""),
			Name:       "test1",
		}
		cli := NewGitUploadClient(repo.Branch)
		require.NoError(t, cli.CreateFiles(
			ctx,
			author,
			repo,
			"hey test",
			[]RepoFile{
				{
					Content: []byte("fake content"),
					Path:    "fake.txt",
				},
				{
					Content: []byte("fake2 content"),
					Path:    "fake2.txt",
				},
				{
					Content: []byte("fake content updated"),
					Path:    "fake.txt",
				},
			},
		))
		require.Equal(t, "fake content updated", string(testlib.CatFileFromBareRepository(t, url, "fake.txt")))
		require.Equal(t, "fake2 content", string(testlib.CatFileFromBareRepository(t, url, "fake2.txt")))
	})

	t.Run("with new branch", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     url,
			PrivateKey: testlib.MakeNewSSHKey(t, ""),
			Name:       "test1",
			Branch:     "new-branch",
		}
		cli := NewGitUploadClient(repo.Branch)
		require.NoError(t, cli.CreateFiles(
			ctx,
			author,
			repo,
			"hey test",
			[]RepoFile{
				{
					Content: []byte("fake content"),
					Path:    "fake.txt",
				},
				{
					Content: []byte("fake2 content"),
					Path:    "fake2.txt",
				},
				{
					Content: []byte("fake content updated"),
					Path:    "fake.txt",
				},
			},
		))
		for path, content := range map[string]string{
			"fake.txt":  "fake content updated",
			"fake2.txt": "fake2 content",
		} {
			require.Equal(
				t, content,
				string(testlib.CatFileFromBareRepositoryOnBranch(
					t, url,
					repo.Branch,
					path,
				)),
			)
		}
	})

	t.Run("no repo name", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     url,
			PrivateKey: testlib.MakeNewSSHKey(t, ""),
		}
		cli := NewGitUploadClient(repo.Branch)
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
	t.Run("clone fail", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:     "git@github.com:nope/nopenopenopenope",
			PrivateKey: testlib.MakeNewSSHKey(t, ""),
		}
		cli := NewGitUploadClient(repo.Branch)
		err := cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte{},
			"filename",
			"msg",
		)
		require.ErrorContains(t, err, "failed to clone")
	})
	t.Run("bad ssh cmd", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Dist: t.TempDir(),
		})
		repo := Repo{
			GitURL:        testlib.GitMakeBareRepository(t),
			PrivateKey:    testlib.MakeNewSSHKey(t, ""),
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
		ctx := testctx.NewWithCfg(config.Project{
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
	t.Run("bad ssh cmd", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
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
		ctx := testctx.NewWithCfg(config.Project{
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
	t.Run("with valid path", func(t *testing.T) {
		path := testlib.MakeNewSSHKey(t, "")
		result, err := keyPath(path)
		require.NoError(t, err)
		require.Equal(t, path, result)
	})
	t.Run("with invalid path", func(t *testing.T) {
		result, err := keyPath("testdata/nope")
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Empty(t, result)
	})

	t.Run("with password protected key path", func(t *testing.T) {
		path := testlib.MakeNewSSHKey(t, "pwd")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(string(bts))
		require.EqualError(t, err, "git: key is password-protected")
		require.Empty(t, result)
	})

	t.Run("with key", func(t *testing.T) {
		for _, algo := range []keygen.KeyType{keygen.Ed25519, keygen.RSA} {
			t.Run(string(algo), func(t *testing.T) {
				path := testlib.MakeNewSSHKeyType(t, "", algo)
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
		require.Empty(t, result)
	})
	t.Run("with invalid EOF", func(t *testing.T) {
		path := testlib.MakeNewSSHKey(t, "")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, err := keyPath(strings.TrimSpace(string(bts)))
		require.NoError(t, err)

		resultbts, err := os.ReadFile(result)
		require.NoError(t, err)
		require.Equal(t, string(bts), string(resultbts))
	})
}

func TestRepoFromURL(t *testing.T) {
	for k, v := range map[string]string{
		"goreleaser": "git@github.com:goreleaser/goreleaser.git",
		"nfpm":       "https://github.com/goreleaser/nfpm",
		"test":       "https://myserver.git/foo/test.git",
	} {
		t.Run(k, func(t *testing.T) {
			require.Equal(t, k, nameFromURL(v))
		})
	}
}
