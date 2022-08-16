package git_test

import (
	"context"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestGit(t *testing.T) {
	ctx := context.Background()
	out, err := git.Run(ctx, "status")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	out, err = git.Run(ctx, "command-that-dont-exist")
	require.Error(t, err)
	require.Empty(t, out)
	require.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n",
		err.Error(),
	)
}

func TestGitWarning(t *testing.T) {
	ctx := context.Background()
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "foo")
	testlib.GitBranch(t, "tags/1.2.2")
	testlib.GitTag(t, "1.2.2")
	testlib.GitCommit(t, "foobar")
	testlib.GitBranch(t, "tags/1.2.3")
	testlib.GitTag(t, "1.2.3")

	out, err := git.Run(ctx, "describe", "--tags", "--abbrev=0", "tags/1.2.3^")
	require.NoError(t, err)
	require.Equal(t, "1.2.2\n", out)
}

func TestRepo(t *testing.T) {
	ctx := context.Background()
	require.True(t, git.IsRepo(ctx), "goreleaser folder should be a git repo")

	require.NoError(t, os.Chdir(os.TempDir()))
	require.False(t, git.IsRepo(ctx), os.TempDir()+" folder should be a git repo")
}

func TestClean(t *testing.T) {
	ctx := context.Background()
	out, err := git.Clean("asdasd 'ssadas'\nadasd", nil)
	require.NoError(t, err)
	require.Equal(t, "asdasd ssadas", out)

	out, err = git.Clean(git.Run(ctx, "command-that-dont-exist"))
	require.Error(t, err)
	require.Empty(t, out)
	require.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.",
		err.Error(),
	)
}
