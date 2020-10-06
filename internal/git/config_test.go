package git_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestRepoName(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")
	repo, err := git.ExtractRepoFromConfig()
	require.NoError(t, err)
	require.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestExtractRepoFromURL(t *testing.T) {
	for _, url := range []string{
		"git@github.com:goreleaser/goreleaser.git",
		"git@custom:goreleaser/goreleaser.git",
		"git@custom:crazy/url/goreleaser/goreleaser.git",
		"https://github.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/crazy/url/goreleaser/goreleaser.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo := git.ExtractRepoFromURL(url)
			require.Equal(t, "goreleaser/goreleaser", repo.String())
		})
	}
}
