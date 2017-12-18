package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestRepoName(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")
	repo, err := remoteRepo()
	assert.NoError(t, err)
	assert.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromGitURL(t *testing.T) {
	repo := extractRepoFromURL("git@github.com:goreleaser/goreleaser.git")
	assert.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestExtractReporFromHttpsURL(t *testing.T) {
	repo := extractRepoFromURL("https://github.com/goreleaser/goreleaser.git")
	assert.Equal(t, "goreleaser/goreleaser", repo.String())
}
