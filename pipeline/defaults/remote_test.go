package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	assert := assert.New(t)
	name, err := remoteRepo()
	assert.NoError(err)
	assert.Equal("goreleaser/goreleaser", name)
}

func TestExtractReporFromGitURL(t *testing.T) {
	assert := assert.New(t)
	url := extractRepoFromURL("git@github.com:goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", url)
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	assert := assert.New(t)
	url := extractRepoFromURL("https://github.com/goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", url)
}
