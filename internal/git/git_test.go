package git_test

import (
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
