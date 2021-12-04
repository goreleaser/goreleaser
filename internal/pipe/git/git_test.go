package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNotAGitFolder(t *testing.T) {
	testlib.Mktmp(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.EqualError(t, Pipe{}.Run(ctx), ErrNotRepository.Error())
}

func TestSingleCommit(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
	require.Equal(t, "commit1", ctx.Git.Subject)
}

func TestAnnotatedTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitAnnotatedTag(t, "v0.0.1", "first version")
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Equal(t, "first version", ctx.Git.Subject)
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
}

func TestBranch(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "test-branch-commit")
	testlib.GitTag(t, "test-branch-tag")
	testlib.GitCheckoutBranch(t, "test-branch")
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "test-branch", ctx.Git.Branch)
	require.Equal(t, "test-branch-tag", ctx.Git.Summary)
}

func TestNoRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.EqualError(t, Pipe{}.Run(ctx), "couldn't get remote URL: fatal: No remote configured to list refs from.")
}

func TestNewRepository(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	ctx := &context.Context{
		Config: config.Project{},
	}
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
	ctx := context.New(config.Project{})
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
		err := Pipe{}.Run(context.New(config.Project{}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "git is currently in a dirty state")
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipValidate = true
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.Snapshot = true
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
}

func TestRemoteURLContainsWithUsernameAndToken(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "https://gitlab-ci-token:SyYhsAghYFTvMoxw7GAg@gitlab.private.com/platform/base/poc/kink.git/releases/tag/v0.1.4")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestRemoteURLContainsWithUsernameAndTokenWithInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "https://gitlab-ci-token:SyYhsAghYFTvMoxw7GAggitlab.com/platform/base/poc/kink.git/releases/tag/v0.1.4")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	ctx := context.New(config.Project{})
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
		require.NoError(t, Pipe{}.Run(context.New(config.Project{})))
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipValidate = true
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.Snapshot = true
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
}

func TestTagIsNotLastCommit(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	ctx := &context.Context{
		Config: config.Project{},
	}
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
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.2")
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
	require.Equal(t, "git@github.com:foo/bar.git", ctx.Git.URL)
}

func TestSnapshotNoTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	ctx := context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo.CurrentTag, ctx.Git.CurrentTag)
	require.Empty(t, ctx.Git.PreviousTag)
}

func TestSnapshotNoCommits(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	ctx := context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotWithoutRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := context.New(config.Project{})
	ctx.Snapshot = true
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
	ctx := context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
}

func TestGitNotInPath(t *testing.T) {
	path := os.Getenv("PATH")
	defer func() {
		require.NoError(t, os.Setenv("PATH", path))
	}()
	require.NoError(t, os.Setenv("PATH", ""))
	require.EqualError(t, Pipe{}.Run(context.New(config.Project{})), ErrNoGit.Error())
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

		ctx := &context.Context{
			Config: config.Project{},
		}
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
	ctx := &context.Context{
		Config: config.Project{},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
	require.Empty(t, ctx.Git.PreviousTag, "should be empty")
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

			ctx := &context.Context{
				Config: config.Project{},
			}
			require.NoError(t, Pipe{}.Run(ctx))
			require.Equal(t, tc.expected, ctx.Git.PreviousTag)

			for name := range tc.envs {
				require.NoError(t, os.Setenv(name, ""))
			}
		})
	}
}
