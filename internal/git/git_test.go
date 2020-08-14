package git_test

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	out, err := git.Run("status")
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	out, err = git.Run("command-that-dont-exist")
	assert.Error(t, err)
	assert.Empty(t, out)
	assert.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n",
		err.Error(),
	)
}

func TestGitWarning(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "foo")
	testlib.GitBranch(t, "tags/1.2.2")
	testlib.GitTag(t, "1.2.2")
	testlib.GitCommit(t, "foobar")
	testlib.GitBranch(t, "tags/1.2.3")
	testlib.GitTag(t, "1.2.3")

	out, err := git.Run("describe", "--tags", "--abbrev=0", "tags/1.2.3^")
	assert.NoError(t, err)
	assert.Equal(t, "1.2.2\n", out)
}

func TestRepo(t *testing.T) {
	assert.True(t, git.IsRepo(), "goreleaser folder should be a git repo")

	assert.NoError(t, os.Chdir(os.TempDir()))
	assert.False(t, git.IsRepo(), os.TempDir()+" folder should be a git repo")
}

func TestClean(t *testing.T) {
	out, err := git.Clean("asdasd 'ssadas'\nadasd", nil)
	assert.NoError(t, err)
	assert.Equal(t, "asdasd ssadas", out)

	out, err = git.Clean(git.Run("command-that-dont-exist"))
	assert.Error(t, err)
	assert.Empty(t, out)
	assert.Equal(
		t,
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.",
		err.Error(),
	)
}
