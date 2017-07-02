package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	var assert = assert.New(t)
	repo, err := remoteRepo()
	assert.NoError(err)
	assert.Equal("goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromGitURL(t *testing.T) {
	var assert = assert.New(t)
	repo := extractRepoFromURL("git@github.com:goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	var assert = assert.New(t)
	repo := extractRepoFromURL("https://github.com/goreleaser/goreleaser.git")
	assert.Equal("goreleaser/goreleaser", repo.String())
}
