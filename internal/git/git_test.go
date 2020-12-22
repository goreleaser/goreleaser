package git_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestGit(t *testing.T) {
	out, err := git.Run("status")
	require.NoError(t, err)
	require.NotEmpty(t, out)

	out, err = git.Run("command-that-dont-exist")
	require.Error(t, err)
	require.Empty(t, out)
	require.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n",
		err.Error(),
	)
}

func TestGitWarning(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "foo")
	testlib.GitBranch(t, "tags/1.2.2")
	testlib.GitTag(t, "1.2.2")
	testlib.GitCommit(t, "foobar")
	testlib.GitBranch(t, "tags/1.2.3")
	testlib.GitTag(t, "1.2.3")

	out, err := git.Run("describe", "--tags", "--abbrev=0", "tags/1.2.3^")
	require.NoError(t, err)
	require.Equal(t, "1.2.2\n", out)
}

func TestRepo(t *testing.T) {
	require.True(t, git.IsRepo(), "goreleaser folder should be a git repo")

	require.NoError(t, os.Chdir(os.TempDir()))
	require.False(t, git.IsRepo(), os.TempDir()+" folder should be a git repo")
}

func TestStatus(t *testing.T) {
	var folder = testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	require.NoError(t, err)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	require.NoError(t, ioutil.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0644))
	status, dirty := git.Status()
	require.True(t, dirty, "git is currently in a dirty state")
	require.NotEmpty(t, status, "git status")
}

func TestClean(t *testing.T) {
	out, err := git.Clean("asdasd 'ssadas'\nadasd", nil)
	require.NoError(t, err)
	require.Equal(t, "asdasd ssadas", out)

	out, err = git.Clean(git.Run("command-that-dont-exist"))
	require.Error(t, err)
	require.Empty(t, out)
	require.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.",
		err.Error(),
	)
}
