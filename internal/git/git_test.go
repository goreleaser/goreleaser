package git_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	out, err := git.Run("init")
	assert.NoError(err)
	assert.Contains(out, "Initialized empty Git repository")

	out, err = git.Run("command-that-dont-exist")
	assert.Error(err)
	assert.Empty(out)
	assert.Equal(
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n",
		err.Error(),
	)
}
