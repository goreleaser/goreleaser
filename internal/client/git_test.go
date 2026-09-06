package client

import (
	"os"
	"os/exec"
	"path/filepath"
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
			GitURL:        "git@localhost:nope/nopenopenopenope",
			PrivateKey:    sshKey,
			GitSSHCommand: `ssh -i "{{ .KeyPath }}" -o StrictHostKeyChecking=accept-new -o ConnectTimeout=1 -o BatchMode=yes -F /dev/null`,
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
		result, cleanup, err := keyPath(sshKey)
		require.NoError(t, err)
		require.Nil(t, cleanup)
		require.Equal(t, sshKey, result)
	})
	t.Run("with invalid path", func(t *testing.T) {
		t.Parallel()
		result, cleanup, err := keyPath("testdata/nope")
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Nil(t, cleanup)
		require.Empty(t, result)
	})

	t.Run("with password protected key path", func(t *testing.T) {
		t.Parallel()
		path := testlib.MakeNewSSHKey(t, "pwd")
		bts, err := os.ReadFile(path)
		require.NoError(t, err)

		result, cleanup, err := keyPath(string(bts))
		require.EqualError(t, err, "git: key is password-protected")
		require.Nil(t, cleanup)
		require.Empty(t, result)
	})

	t.Run("with key", func(t *testing.T) {
		t.Parallel()

		_, cleanup, err := keyPath(sshKey)
		require.NoError(t, err)
		require.Nil(t, cleanup)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		result, cleanup, err := keyPath("")
		require.EqualError(t, err, `private_key is empty`)
		require.Nil(t, cleanup)
		require.Empty(t, result)
	})

	t.Run("with invalid EOF", func(t *testing.T) {
		t.Parallel()
		bts, err := os.ReadFile(sshKey)
		require.NoError(t, err)

		result, cleanup, err := keyPath(strings.TrimSpace(string(bts)))
		require.NoError(t, err)
		require.NotNil(t, cleanup)

		resultbts, err := os.ReadFile(result)
		require.NoError(t, err)
		require.Equal(t, string(bts), string(resultbts))
		require.NoError(t, cleanup())
		require.NoFileExists(t, result)
	})
}

func TestGitClientRemovesInlinePrivateKey(t *testing.T) {
	author := config.CommitAuthor{
		Name:  "Foo",
		Email: "foo@bar.com",
	}
	sshKey := testlib.MakeNewSSHKey(t, "")
	bts, err := os.ReadFile(sshKey)
	require.NoError(t, err)
	inlineKey := string(bts)

	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)

	requireNoGeneratedKeys := func(tb testing.TB) {
		tb.Helper()
		entries, err := os.ReadDir(temp)
		require.NoError(tb, err)
		for _, entry := range entries {
			require.False(tb, strings.HasPrefix(entry.Name(), "id_"), "generated key %q was not removed", entry.Name())
		}
	}

	t.Run("success", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: inlineKey,
			Name:       "inline-key-success",
		}
		cli := NewGitUploadClient(repo.Branch)
		require.NoError(t, cli.CreateFile(ctx, author, repo, []byte("content"), "file.txt", "add file"))
		requireNoGeneratedKeys(t)
	})

	t.Run("clone failure", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:        "git@localhost:nope/nopenopenopenope",
			PrivateKey:    inlineKey,
			GitSSHCommand: `ssh -i "{{ .KeyPath }}" -o StrictHostKeyChecking=accept-new -o ConnectTimeout=1 -o BatchMode=yes -F /dev/null`,
		}
		cli := NewGitUploadClient(repo.Branch)
		err := cli.CreateFile(ctx, author, repo, []byte{}, "filename", "msg")
		require.ErrorContains(t, err, "failed to clone")
		requireNoGeneratedKeys(t)
	})

	t.Run("ssh command template failure", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:        testlib.GitMakeBareRepository(t),
			PrivateKey:    inlineKey,
			GitSSHCommand: "{{.Foo}}",
		}
		cli := NewGitUploadClient(repo.Branch)
		testlib.RequireTemplateError(t, cli.CreateFile(ctx, author, repo, []byte{}, "filename", "msg"))
		requireNoGeneratedKeys(t)
	})

	t.Run("caller-owned path", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist: t.TempDir(),
		})

		repo := Repo{
			GitURL:     url,
			PrivateKey: sshKey,
			Name:       "caller-owned-key",
		}
		cli := NewGitUploadClient(repo.Branch)
		require.NoError(t, cli.CreateFile(ctx, author, repo, []byte("content"), "file.txt", "add file"))
		require.FileExists(t, sshKey)
		requireNoGeneratedKeys(t)
	})
}

func TestGitClientUsesRepositorySpecificCheckout(t *testing.T) {
	sshKey := testlib.MakeNewSSHKey(t, "")
	author := config.CommitAuthor{
		Name:  "Foo",
		Email: "foo@bar.com",
	}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: t.TempDir(),
	})
	cli := NewGitUploadClient("main")

	first := testlib.GitMakeBareRepository(t)
	second := testlib.GitMakeBareRepository(t)
	repo := Repo{
		PrivateKey: sshKey,
		Name:       "homebrew-tap",
		Branch:     "main",
	}

	repo.GitURL = first
	require.NoError(t, cli.CreateFile(ctx, author, repo, []byte("first"), "first.rb", "add first"))

	repo.GitURL = second
	require.NoError(t, cli.CreateFile(ctx, author, repo, []byte("second"), "second.rb", "add second"))

	require.Equal(t, "first.rb", gitInBare(t, first, "ls-tree", "--name-only", "-r", "main"))
	require.Equal(t, "second.rb", gitInBare(t, second, "ls-tree", "--name-only", "-r", "main"))
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

func TestCommitConfigFlags(t *testing.T) {
	t.Parallel()
	base := func(extra ...string) []string {
		return append([]string{
			"-c", "user.name=Foo",
			"-c", "user.email=foo@bar.com",
		}, extra...)
	}
	for _, tt := range []struct {
		name     string
		signing  config.CommitSigning
		expected []string
	}{
		{
			name:     "disabled",
			expected: base("-c", "commit.gpgSign=false"),
		},
		{
			name: "disabled ignores the other options",
			signing: config.CommitSigning{
				Key:     "test-signing-key",
				Program: "/usr/bin/gpg",
				Format:  "ssh",
			},
			expected: base("-c", "commit.gpgSign=false"),
		},
		{
			name:     "enabled with no options",
			signing:  config.CommitSigning{Enabled: true},
			expected: base("-c", "commit.gpgSign=true"),
		},
		{
			name: "enabled with all options",
			signing: config.CommitSigning{
				Enabled: true,
				Key:     "test-signing-key",
				Program: "/usr/bin/gpg",
				Format:  "ssh",
			},
			expected: base(
				"-c", "commit.gpgSign=true",
				"-c", "user.signingKey=test-signing-key",
				"-c", "gpg.program=/usr/bin/gpg",
				"-c", "gpg.format=ssh",
			),
		},
		{
			name:    "enabled with the default format",
			signing: config.CommitSigning{Enabled: true, Format: "openpgp"},
			expected: base(
				"-c", "commit.gpgSign=true",
				"-c", "gpg.format=openpgp",
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, commitConfigFlags(config.CommitAuthor{
				Name:    "Foo",
				Email:   "foo@bar.com",
				Signing: tt.signing,
			}))
		})
	}
}

// gitInBare runs git against a bare repository. It uses --git-dir because
// -C is rejected when safe.bareRepository is set to explicit.
func gitInBare(tb testing.TB, bare string, args ...string) string {
	tb.Helper()
	out, err := exec.CommandContext(
		tb.Context(),
		"git",
		append([]string{"--git-dir=" + bare}, args...)...,
	).CombinedOutput()
	require.NoError(tb, err, string(out))
	return strings.TrimSpace(string(out))
}

func TestGitClientReconfiguresReusedCheckout(t *testing.T) {
	sshKey := testlib.MakeNewSSHKey(t, "")

	t.Run("commit author", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Dist: t.TempDir()})
		repo := Repo{GitURL: url, PrivateKey: sshKey, Name: "reused-author"}
		cli := NewGitUploadClient(repo.Branch)

		require.NoError(t, cli.CreateFile(ctx, config.CommitAuthor{
			Name: "First", Email: "first@example.com",
		}, repo, []byte("first"), "file.txt", "first"))
		require.NoError(t, cli.CreateFile(ctx, config.CommitAuthor{
			Name: "Second", Email: "second@example.com",
		}, repo, []byte("second"), "file.txt", "second"))

		require.Equal(
			t,
			"Second <second@example.com>",
			gitInBare(t, url, "log", "master", "-1", "--format=%an <%ae>"),
		)
	})

	t.Run("signing does not leak to the next caller", func(t *testing.T) {
		url := testlib.GitMakeBareRepository(t)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{Dist: t.TempDir()})
		repo := Repo{GitURL: url, PrivateKey: sshKey, Name: "reused-signing"}
		cli := NewGitUploadClient(repo.Branch)
		// A program that cannot exist, so that the result does not depend on a
		// gpg installation or on the keyring of whoever runs the tests.
		const programName = "not-gpg"
		program := filepath.Join(t.TempDir(), programName)

		require.NoError(t, cli.CreateFile(ctx, config.CommitAuthor{
			Name: "First", Email: "first@example.com",
		}, repo, []byte("first"), "file.txt", "first"))

		err := cli.CreateFile(ctx, config.CommitAuthor{
			Name:  "Second",
			Email: "second@example.com",
			Signing: config.CommitSigning{
				Enabled: true,
				Key:     "test-signing-key",
				Program: program,
			},
		}, repo, []byte("second"), "file.txt", "second")
		// git names the program it could not run, which shows that gpg.program
		// reached it. Only the base name is matched, because git rewrites path
		// separators on Windows.
		require.ErrorContains(t, err, programName)

		require.NoError(t, cli.CreateFile(ctx, config.CommitAuthor{
			Name: "Third", Email: "third@example.com",
		}, repo, []byte("third"), "file.txt", "third"))

		require.Equal(
			t,
			"Third <third@example.com>",
			gitInBare(t, url, "log", "master", "-1", "--format=%an <%ae>"),
		)
		require.NotContains(
			t,
			gitInBare(t, url, "cat-file", "commit", "master"),
			"gpgsig",
			"the third commit must not be signed",
		)
	})
}

func TestRepoFromURL(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		url  string
	}{
		{"goreleaser", "git@github.com:goreleaser/goreleaser.git"},
		{"nfpm", "https://github.com/goreleaser/nfpm"},
		{"test", "https://myserver.git/foo/test.git"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.name, nameFromURL(tt.url))
		})
	}
}
