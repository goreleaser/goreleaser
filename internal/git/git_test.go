package git_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	gitpipe "github.com/goreleaser/goreleaser/v2/internal/pipe/git"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestGitRemoteCredentials(t *testing.T) {
	const remote = "https://synthetic-user:synthetic-password@example.invalid/owner/repo.git"
	for _, rewrite := range []bool{false, true} {
		name := "stored"
		if rewrite {
			name = "insteadOf"
		}
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_COUNT", "0")
			t.Setenv("GIT_CONFIG_PARAMETERS", "")
			testlib.GitInit(t)
			testlib.GitCommit(t, "initial")
			testlib.GitTag(t, "v1.0.0")
			if rewrite {
				testlib.GitRemoteAdd(t, "git@example.invalid:owner/repo.git")
				_, err := git.Run(t.Context(), "config",
					"url.https://synthetic-user:synthetic-password@example.invalid/.insteadOf",
					"git@example.invalid:")
				require.NoError(t, err)
			} else {
				testlib.GitRemoteAdd(t, remote)
			}

			var logs bytes.Buffer
			previous := log.Log
			log.Log = log.New(&logs)
			log.SetLevel(log.DebugLevel)
			t.Cleanup(func() { log.Log = previous })

			out, err := git.Run(t.Context(), "ls-remote", "--get-url")
			require.NoError(t, err)
			require.Equal(t, remote+"\n", out)

			ctx := testctx.Wrap(t.Context())
			require.NoError(t, gitpipe.Pipe{}.Run(ctx))
			require.Equal(t, "https://example.invalid/owner/repo.git", ctx.Git.URL)
			require.Contains(t, logs.String(), "git command result")
			require.Contains(t, logs.String(), "ls-remote --get-url")
			require.NotContains(t, logs.String(), "synthetic-user")
			require.NotContains(t, logs.String(), "synthetic-password")
			require.Contains(t, logs.String(), "https://example.invalid/owner/repo.git")
		})
	}
}

func TestGitCredentialDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name string
		url  string
		want string
	}{
		{
			name: "username and password",
			url:  "https://synthetic-user:synthetic-password@example.invalid/repo.git",
			want: "https://example.invalid/repo.git",
		},
		{
			name: "token username",
			url:  "https://synthetic-token@example.invalid/repo.git",
			want: "https://example.invalid/repo.git",
		},
		{
			name: "percent encoded",
			url:  "https://synthetic%2Duser:synthetic%40password@example.invalid/repo.git",
			want: "https://example.invalid/repo.git",
		},
		{
			name: "ssh and ipv6",
			url:  "ssh://synthetic-user:synthetic-password@[::1]:2222/repo.git",
			want: "ssh://[::1]:2222/repo.git",
		},
		{
			name: "without credentials",
			url:  "https://example.invalid/repo.git",
			want: "https://example.invalid/repo.git",
		},
		{
			name: "scheme stripped by git",
			url:  "//synthetic-user:synthetic-password@D:/tmp/repo.git",
			want: "//D:/tmp/repo.git",
		},
		{
			name: "scp remote",
			url:  "git@example.invalid:repo.git",
			want: "git@example.invalid:repo.git",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_COUNT", "0")
			t.Setenv("GIT_CONFIG_PARAMETERS", "")
			t.Setenv("LC_ALL", "C")
			testlib.GitInit(t)

			var logs bytes.Buffer
			previous := log.Log
			log.Log = log.New(&logs)
			log.SetLevel(log.DebugLevel)
			t.Cleanup(func() { log.Log = previous })

			out, err := git.Run(t.Context(), "remote", "get-url", tt.url)
			require.Empty(t, out)
			require.EqualError(t, err, "error: No such remote '"+tt.want+"'\n")
			require.Contains(t, logs.String(), "No such remote")
			require.Contains(t, logs.String(), tt.want)
			require.NotContains(t, logs.String(), "synthetic")
		})
	}
}

func TestGit(t *testing.T) {
	ctx := t.Context()
	out, err := git.Run(ctx, "status")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	out, err = git.Run(ctx, "command-that-dont-exist")
	require.EqualError(t, err, "git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n")
	require.Empty(t, out)
}

func TestGitWarning(t *testing.T) {
	ctx := t.Context()
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "foo")
	testlib.GitBranch(t, "tags/1.2.2")
	testlib.GitTag(t, "1.2.2")
	testlib.GitCommit(t, "foobar")
	testlib.GitBranch(t, "tags/1.2.3")
	testlib.GitTag(t, "1.2.3")
	testlib.GitTag(t, "nightly")

	out, err := git.Run(ctx, "describe", "--tags", "--abbrev=0", "tags/1.2.3^")
	require.NoError(t, err)
	require.Equal(t, "1.2.2\n", out)

	tags, err := git.CleanAllLines(git.Run(ctx, "tag", "--points-at", "HEAD", "--sort", "-version:refname"))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.2.3", "nightly"}, tags)
}

func TestRepo(t *testing.T) {
	ctx := t.Context()
	require.True(t, git.IsRepo(ctx), "goreleaser folder should be a git repo")

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	require.False(t, git.IsRepo(ctx), tmpDir+" folder should be a git repo")
}

func TestClean(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		out, err := git.Clean("asdasd 'ssadas'\nadasd", nil)
		require.NoError(t, err)
		require.Equal(t, "asdasd ssadas", out)
	})

	t.Run("error", func(t *testing.T) {
		out, err := git.Clean(git.Run(ctx, "command-that-dont-exist"))
		require.EqualError(t, err, "git: 'command-that-dont-exist' is not a git command. See 'git --help'.")
		require.Empty(t, out)
	})

	t.Run("all lines error", func(t *testing.T) {
		out, err := git.CleanAllLines(git.Run(ctx, "command-that-dont-exist"))
		require.EqualError(t, err, "git: 'command-that-dont-exist' is not a git command. See 'git --help'.")
		require.Empty(t, out)
	})
}

func TestGitColumnUI(t *testing.T) {
	ctx := t.Context()
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "foo")
	testlib.GitTag(t, "1.2.3")
	testlib.GitTag(t, "nightly")
	_, err := git.Run(ctx, "config", "column.ui", "always")
	require.NoError(t, err)

	tags, err := git.CleanAllLines(git.Run(ctx, "tag", "--points-at", "HEAD", "--sort", "-version:refname"))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1.2.3", "nightly"}, tags)

	out, err := git.Run(ctx, "tag", "-l", "--format=%(refname:short)%00%(objecttype)", "1.2.3")
	require.NoError(t, err)
	require.Equal(t, "1.2.3\x00commit\n", out)
}
