package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommit(t *testing.T) {
	assert := assert.New(t)
	commit, err := commitHash()
	assert.NoError(err)
	assert.NotEmpty(commit)
	assert.NotContains(commit, "'")
}
