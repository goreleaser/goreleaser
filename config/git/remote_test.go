package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	assert := assert.New(t)
	name, err := RemoteRepoName()
	assert.NoError(err)
	assert.Equal("goreleaser/releaser", name)
}

func TestExtractReporFromGitURL(t *testing.T) {
	assert := assert.New(t)
	url := extractRepoFromURL("git@github.com:goreleaser/releaser.git")
	assert.Equal("goreleaser/releaser", url)
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	assert := assert.New(t)
	url := extractRepoFromURL("https://github.com/goreleaser/releaser.git")
	assert.Equal("goreleaser/releaser", url)
}
