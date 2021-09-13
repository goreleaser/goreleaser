package git_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestRepoName(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")
	repo, err := git.ExtractRepoFromConfig()
	require.NoError(t, err)
	require.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestRepoNameWithDifferentRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAddWithName(t, "upstream", "https://github.com/goreleaser/goreleaser.git")
	_, err := git.Run("pull", "upstream", "master")
	require.NoError(t, err)
	_, err = git.Run("branch", "--set-upstream-to", "upstream/master")
	require.NoError(t, err)
	repo, err := git.ExtractRepoFromConfig()
	require.NoError(t, err)
	require.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestExtractRepoFromURL(t *testing.T) {
	// valid urls
	for _, url := range []string{
		"git@github.com:goreleaser/goreleaser.git",
		"git@custom:goreleaser/goreleaser.git",
		"git@custom:crazy/url/goreleaser/goreleaser.git",
		"https://github.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/goreleaser/goreleaser.git",
		"https://github.enterprise.com/crazy/url/goreleaser/goreleaser.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo, err := git.ExtractRepoFromURL(url)
			require.NoError(t, err)
			require.Equal(t, "goreleaser/goreleaser", repo.String())
		})
	}

	// invalid urls
	for _, url := range []string{
		"git@gist.github.com:someid.git",
		"https://gist.github.com/someid.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo, err := git.ExtractRepoFromURL(url)
			require.EqualError(t, err, "unsupported repository URL: "+url)
			require.Equal(t, "", repo.String())
		})
	}
}
