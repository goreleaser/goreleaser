//go:build integration

package git_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRelativeRemote(t *testing.T) {
	ctx := t.Context()
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAddWithName(t, "upstream", "https://github.com/goreleaser/goreleaser.git")
	_, err := git.Run(ctx, "pull", "upstream", "main")
	require.NoError(t, err)
	_, err = git.Run(ctx, "branch", "--set-upstream-to", "upstream/main")
	require.NoError(t, err)
	_, err = git.Run(ctx, "checkout", "--track", "-b", "relative_branch")
	require.NoError(t, err)
	gitCfg, err := git.Run(ctx, "config", "--local", "--list")
	require.NoError(t, err)
	require.Contains(t, gitCfg, "branch.relative_branch.remote=.")
	repo, err := git.ExtractRepoFromConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "goreleaser/goreleaser", repo.String())
}

func TestIntegrationRepoNameWithDifferentRemote(t *testing.T) {
	ctx := t.Context()
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAddWithName(t, "upstream", "https://github.com/goreleaser/goreleaser.git")
	_, err := git.Run(ctx, "pull", "upstream", "main")
	require.NoError(t, err)
	_, err = git.Run(ctx, "branch", "--set-upstream-to", "upstream/main")
	require.NoError(t, err)
	repo, err := git.ExtractRepoFromConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "goreleaser/goreleaser", repo.String())
}
