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

	t.Run("full", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
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
	t.Run("clone fail", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     "git@localhost:nope/nopenopenopenope",
			PrivateKey: sshKey,
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

func TestGitClientWithSigning(t *testing.T) {
	t.Parallel()

	sshKey := testlib.MakeNewSSHKey(t, "")

	t.Run("commit signing enabled", func(t *testing.T) {
		t.Parallel()
		author := config.CommitAuthor{
			Name:  "Foo",
			Email: "foo@bar.com",
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ABC123DEF456",
				Program: "/usr/bin/gpg",
				Format:  "openpgp",
			},
		}

		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
			Name:       "test-signing",
		}
		cli := NewGitUploadClient(repo.Branch)

		err := cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("test content with signing"),
			"signed.txt",
			"test signed commit",
		)
		require.ErrorContains(t, err, "gpg")
	})

	t.Run("commit signing disabled", func(t *testing.T) {
		t.Parallel()
		author := config.CommitAuthor{
			Name:  "Foo",
			Email: "foo@bar.com",
			Signing: config.CommitSigning{
				Enabled: false,
			},
		}

		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
			Name:       "test-no-signing",
		}
		cli := NewGitUploadClient(repo.Branch)

		require.NoError(t, cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("test content without signing"),
			"unsigned.txt",
			"test unsigned commit",
		))
	})

	t.Run("commit signing with ssh format", func(t *testing.T) {
		t.Parallel()
		author := config.CommitAuthor{
			Name:  "Foo",
			Email: "foo@bar.com",
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG...",
				Format:  "ssh",
			},
		}

		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
			Name:       "test-ssh-signing",
		}
		cli := NewGitUploadClient(repo.Branch)

		err := cli.CreateFile(
			ctx,
			author,
			repo,
			[]byte("test content with ssh signing"),
			"ssh-signed.txt",
			"test ssh signed commit",
		)
		if testlib.IsWindows() {
			require.Error(t, err)
			return
		}
		require.ErrorContains(t, err, "public key")
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
