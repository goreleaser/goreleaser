//go:build integration

package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestIntegrationGitClient(t *testing.T) {
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
}

func TestIntegrationGitClientWithSigning(t *testing.T) {
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
