package git_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestNotARepo(t *testing.T) {
	testlib.Mktmp(t)
	_, err := git.ExtractRepoFromConfig()
	require.EqualError(t, err, `current folder is not a git repository`)
}

func TestNoRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	_, err := git.ExtractRepoFromConfig()
	require.EqualError(t, err, `no remote configured to list refs from`)
}

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
	_, err := git.Run("pull", "upstream", "main")
	require.NoError(t, err)
	_, err = git.Run("branch", "--set-upstream-to", "upstream/main")
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
		"https://foo@github.com/goreleaser/goreleaser",
		"https://github.com/goreleaser/goreleaser.git",
		"https://something.with.port:8080/goreleaser/goreleaser.git",
		"https://github.enterprise.com/goreleaser/goreleaser.git",
		"https://gitlab-ci-token:SOME_TOKEN@gitlab.yourcompany.com/goreleaser/goreleaser.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo, err := git.ExtractRepoFromURL(url)
			require.NoError(t, err)
			require.Equal(t, "goreleaser/goreleaser", repo.String())
		})
	}

	// nested urls
	for _, url := range []string{
		"git@custom:group/nested/goreleaser/goreleaser.git",
		"https://gitlab.mycompany.com/group/nested/goreleaser/goreleaser.git",
		"https://gitlab-ci-token:SOME_TOKEN@gitlab.yourcompany.com/group/nested/goreleaser/goreleaser.git",
	} {
		t.Run(url, func(t *testing.T) {
			repo, err := git.ExtractRepoFromURL(url)
			require.NoError(t, err)
			require.Equal(t, "group/nested/goreleaser/goreleaser", repo.String())
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
