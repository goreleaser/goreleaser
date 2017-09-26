package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGit(t *testing.T) {
	var assert = assert.New(t)
	out, err := Run("status")
	assert.NoError(err)
	assert.Contains(out, "On branch")

	out, err = Run("command-that-dont-exist")
	assert.Error(err)
	assert.Empty(out)
	assert.Equal(
		"git: 'command-that-dont-exist' is not a git command. See 'git --help'.\n",
		err.Error(),
	)
}
