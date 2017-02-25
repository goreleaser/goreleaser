package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	assert := assert.New(t)
	repo, err := remoteRepo()
	assert.NoError(err)
	assert.Equal("goreleaser", repo.Owner)
	assert.Equal("goreleaser", repo.Name)
	assert.Equal("github", repo.Provider)
}

func TestExtractReporFromGitURL(t *testing.T) {
	assert := assert.New(t)
	repo := extractRepoFromURL("git@github.com:owner/repo.git")
	assert.Equal("owner", repo.Owner)
	assert.Equal("repo", repo.Name)
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	assert := assert.New(t)
	repo := extractRepoFromURL("https://github.com/owner/repo.git")
	assert.Equal("owner", repo.Owner)
	assert.Equal("repo", repo.Name)
}
