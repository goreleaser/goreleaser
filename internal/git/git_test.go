package git_test

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

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
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	t.Run("work tree", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		isRepo, err := git.IsRepo(t.Context())
		require.NoError(t, err)
		require.True(t, isRepo)
	})

	t.Run("not a repository", func(t *testing.T) {
		testlib.Mktmp(t)
		isRepo, err := git.IsRepo(t.Context())
		require.ErrorContains(t, err, "not a git repository")
		require.False(t, isRepo)
	})

	t.Run("bare repository", func(t *testing.T) {
		dir := testlib.GitMakeBareRepository(t)
		t.Chdir(dir)
		t.Setenv("GIT_DIR", dir)
		isRepo, err := git.IsRepo(t.Context())
		require.NoError(t, err)
		require.False(t, isRepo)
	})

	t.Run("unsafe repository", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		// Exercise Git's ownership check without changing filesystem ownership.
		t.Setenv("GIT_TEST_ASSUME_DIFFERENT_OWNER", "1")
		isRepo, err := git.IsRepo(t.Context())
		require.ErrorContains(t, err, "fatal: detected dubious ownership")
		require.ErrorContains(t, err, "git config --global --add safe.directory")
		require.False(t, isRepo)
	})
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
