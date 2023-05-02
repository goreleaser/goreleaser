package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNotAGitFolder(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New()
	require.EqualError(t, Pipe{}.Run(ctx), ErrNotRepository.Error())
}

func TestSingleCommit(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
	require.Equal(t, "commit1", ctx.Git.TagSubject)
	require.Equal(t, "commit1", ctx.Git.TagContents)
	require.NotEmpty(t, ctx.Git.FirstCommit)
}

func TestAnnotatedTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitAnnotatedTag(t, "v0.0.1", "first version\n\nlalalla\nlalal\nlah")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Equal(t, "first version", ctx.Git.TagSubject)
	require.Equal(t, "first version\n\nlalalla\nlalal\nlah", ctx.Git.TagContents)
	require.Equal(t, "lalalla\nlalal\nlah", ctx.Git.TagBody)
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
}

func TestBranch(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "test-branch-commit")
	testlib.GitTag(t, "test-branch-tag")
	testlib.GitCheckoutBranch(t, "test-branch")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "test-branch", ctx.Git.Branch)
	require.Equal(t, "test-branch-tag", ctx.Git.Summary)
}

func TestNoRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New()
	require.EqualError(t, Pipe{}.Run(ctx), "couldn't get remote URL: fatal: No remote configured to list refs from.")
}

func TestNewRepository(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	ctx := testctx.New()
	// TODO: improve this error handling
	require.Contains(t, Pipe{}.Run(ctx).Error(), `fatal: ambiguous argument 'HEAD'`)
}

// TestNoTagsNoSnapshot covers the situation where a repository
// only contains simple commits and no tags. In this case you have
// to set the --snapshot flag otherwise an error is returned.
func TestNoTagsNoSnapshot(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "first")
	ctx := testctx.New()
	ctx.Snapshot = false
	require.EqualError(t, Pipe{}.Run(ctx), `git doesn't contain any tags. Either add a tag or use --snapshot`)
}

func TestDirty(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	require.NoError(t, err)
	require.NoError(t, dummy.Close())
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	require.NoError(t, os.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0o644))
	t.Run("all checks up", func(t *testing.T) {
		err := Pipe{}.Run(testctx.New())
		require.Error(t, err)
		require.Contains(t, err.Error(), "git is in a dirty state")
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := testctx.New(testctx.SkipValidate)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
		require.True(t, ctx.Git.Dirty)
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.New(testctx.Snapshot)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
		require.True(t, ctx.Git.Dirty)
	})
}

func TestRemoteURLContainsWithUsernameAndToken(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "https://gitlab-ci-token:SyYhsAghYFTvMoxw7GAg@gitlab.private.com/platform/base/poc/kink.git/releases/tag/v0.1.4")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestRemoteURLContainsWithUsernameAndTokenWithInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "https://gitlab-ci-token:SyYhsAghYFTvMoxw7GAggitlab.com/platform/base/poc/kink.git/releases/tag/v0.1.4")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New()
	require.Error(t, Pipe{}.Run(ctx))
}

func TestShallowClone(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(
		t,
		exec.Command(
			"git", "clone",
			"--depth", "1",
			"--branch", "v0.160.0",
			"https://github.com/goreleaser/goreleaser",
			folder,
		).Run(),
	)
	t.Run("all checks up", func(t *testing.T) {
		// its just a warning now
		require.NoError(t, Pipe{}.Run(testctx.New()))
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := testctx.New(testctx.SkipValidate)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.New(testctx.Snapshot)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
}

func TestTagSortOrder(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitCommit(t, "commit2")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.NewWithCfg(config.Project{
		Git: config.Git{
			TagSort: "-version:refname",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
}

func TestTagSortOrderPrerelease(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitCommit(t, "commit2")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1-rc.2")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.NewWithCfg(config.Project{
		Git: config.Git{
			TagSort:          "-version:refname",
			PrereleaseSuffix: "-",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
}

func TestTagIsNotLastCommit(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	ctx := testctx.New()
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "git tag v0.0.1 was not made against commit")
	require.Contains(t, ctx.Git.Summary, "v0.0.1-1-g") // commit not represented because it changes every test
}

func TestValidState(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.3")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.2", ctx.Git.PreviousTag)
	require.Equal(t, "v0.0.3", ctx.Git.CurrentTag)
	require.Equal(t, "git@github.com:foo/bar.git", ctx.Git.URL)
	require.NotEmpty(t, ctx.Git.FirstCommit)
	require.False(t, ctx.Git.Dirty)
}

func TestSnapshotNoTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	ctx := testctx.New(testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo.CurrentTag, ctx.Git.CurrentTag)
	require.Empty(t, ctx.Git.PreviousTag)
	require.NotEmpty(t, ctx.Git.FirstCommit)
}

func TestSnapshotNoCommits(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	ctx := testctx.New(testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotWithoutRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.New(testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotDirty(t *testing.T) {
	folder := testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.0.1")
	require.NoError(t, os.WriteFile(filepath.Join(folder, "foo"), []byte("foobar"), 0o644))
	ctx := testctx.New(testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
}

func TestGitNotInPath(t *testing.T) {
	path := os.Getenv("PATH")
	defer func() {
		require.NoError(t, os.Setenv("PATH", path))
	}()
	require.NoError(t, os.Setenv("PATH", ""))
	require.EqualError(t, Pipe{}.Run(testctx.New()), ErrNoGit.Error())
}

func TestTagFromCI(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitTag(t, "v0.0.2")

	for _, tc := range []struct {
		envs     map[string]string
		expected string
	}{
		{expected: "v0.0.2"},
		{
			envs:     map[string]string{"GORELEASER_CURRENT_TAG": "v0.0.2"},
			expected: "v0.0.2",
		},
	} {
		for name, value := range tc.envs {
			require.NoError(t, os.Setenv(name, value))
		}

		ctx := testctx.New()
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, tc.expected, ctx.Git.CurrentTag)

		for name := range tc.envs {
			require.NoError(t, os.Setenv(name, ""))
		}
	}
}

func TestNoPreviousTag(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Empty(t, ctx.Git.PreviousTag, "should be empty")
	require.NotEmpty(t, ctx.Git.FirstCommit, "should not be empty")
}

func TestPreviousTagFromCI(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.2")

	for _, tc := range []struct {
		envs     map[string]string
		expected string
	}{
		{expected: "v0.0.1"},
		{
			envs:     map[string]string{"GORELEASER_PREVIOUS_TAG": "v0.0.2"},
			expected: "v0.0.2",
		},
	} {
		t.Run(tc.expected, func(t *testing.T) {
			for name, value := range tc.envs {
				require.NoError(t, os.Setenv(name, value))
			}

			ctx := testctx.New()
			require.NoError(t, Pipe{}.Run(ctx))
			require.Equal(t, tc.expected, ctx.Git.PreviousTag)

			for name := range tc.envs {
				require.NoError(t, os.Setenv(name, ""))
			}
		})
	}
}
