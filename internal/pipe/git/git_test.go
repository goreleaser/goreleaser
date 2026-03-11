package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNotAGitFolder(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context())
	require.EqualError(t, Pipe{}.Run(ctx), ErrNotRepository.Error())
}

func TestSingleCommit(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.Wrap(t.Context())
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
	ctx := testctx.Wrap(t.Context())
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
	testlib.GitTag(t, "v1.0.0")
	testlib.GitCheckoutBranch(t, "test-branch")
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "test-branch", ctx.Git.Branch)
	require.Equal(t, "v1.0.0", ctx.Git.Summary)
}

func TestNoRemote(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.Wrap(t.Context())
	require.EqualError(t, Pipe{}.Run(ctx), "couldn't get remote URL: fatal: No remote configured to list refs from.")
}

func TestNewRepository(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	ctx := testctx.Wrap(t.Context())
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
	ctx := testctx.Wrap(t.Context())
	ctx.Snapshot = false
	require.ErrorIs(t, Pipe{}.Run(ctx), ErrNoTag)
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
		err := Pipe{}.Run(testctx.Wrap(t.Context()))
		require.ErrorContains(t, err, "git is in a dirty state")
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Validate))
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
		require.True(t, ctx.Git.Dirty)
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
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
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestRemoteURLContainsWithUsernameAndTokenWithInvalidURL(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "https://gitlab-ci-token:SyYhsAghYFTvMoxw7GAggitlab.com/platform/base/poc/kink.git/releases/tag/v0.1.4")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.Wrap(t.Context())
	require.Error(t, Pipe{}.Run(ctx))
}

func TestShallowClone(t *testing.T) {
	folder := testlib.Mktmp(t)
	require.NoError(
		t,
		exec.CommandContext(
			t.Context(),
			"git", "clone",
			"--depth", "1",
			"--branch", "v0.160.0",
			"https://github.com/goreleaser/goreleaser",
			folder,
		).Run(),
	)
	t.Run("all checks up", func(t *testing.T) {
		// its just a warning now
		require.NoError(t, Pipe{}.Run(testctx.Wrap(t.Context())))
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Validate))
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
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
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	ctx := testctx.Wrap(t.Context())
	err := Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "git tag v0.0.1 was not made against commit")
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
	ctx := testctx.Wrap(t.Context())
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
	ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo.CurrentTag, ctx.Git.CurrentTag)
	require.Empty(t, ctx.Git.PreviousTag)
	require.NotEmpty(t, ctx.Git.FirstCommit)
}

func TestSnapshotNoCommits(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotWithoutRepo(t *testing.T) {
	testlib.Mktmp(t)
	ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
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
	ctx := testctx.Wrap(t.Context(), testctx.Snapshot)
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.1", ctx.Git.Summary)
}

func TestGitNotInPath(t *testing.T) {
	t.Setenv("PATH", "")
	require.EqualError(t, Pipe{}.Run(testctx.Wrap(t.Context())), ErrNoGit.Error())
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
			t.Setenv(name, value)
		}

		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, tc.expected, ctx.Git.CurrentTag)
	}
}

func TestEnvTagsIgnored(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitTag(t, "v0.0.3")

	t.Setenv("GORELEASER_CURRENT_TAG", "v0.0.2")
	t.Setenv("GORELEASER_PREVIOUS_TAG", "v0.0.2")

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Git: config.Git{
			IgnoreTags: []string{"v0.0.2"},
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "v0.0.3", ctx.Git.CurrentTag)
	require.Equal(t, "v0.0.1", ctx.Git.PreviousTag)
}

func TestNoPreviousTag(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	ctx := testctx.Wrap(t.Context())
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
				t.Setenv(name, value)
			}

			ctx := testctx.Wrap(t.Context())
			require.NoError(t, Pipe{}.Run(ctx))
			require.Equal(t, tc.expected, ctx.Git.PreviousTag)
		})
	}
}

func TestFilterTags(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "middle commit")
	testlib.GitTag(t, "nightly")
	testlib.GitCommit(t, "commit2")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitTag(t, "v0.1.0-dev")

	t.Run("no filter", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "nightly", ctx.Git.PreviousTag)
		require.Equal(t, "v0.1.0-dev", ctx.Git.CurrentTag)
	})

	t.Run("template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Git: config.Git{
				IgnoreTags: []string{
					"{{.Env.IGNORE}}",
					"v0.0.2",
					"nightly",
				},
			},
		}, testctx.WithEnv(map[string]string{
			"IGNORE": `v0.0.1`,
		}))

		require.NoError(t, Pipe{}.Run(ctx))
		require.Empty(t, ctx.Git.PreviousTag)
		require.Equal(t, "v0.1.0-dev", ctx.Git.CurrentTag)
	})

	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Git: config.Git{
				IgnoreTags: []string{
					"{{.Env.Nope}}",
				},
			},
		})

		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func TestSemverOnGitPipe(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v1.2.3")
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      2,
		Patch:      3,
		Prerelease: "",
	}, ctx.Semver)
}

func TestSemverOnGitPipePrerelease(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v1.2.3-rc1")
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      2,
		Patch:      3,
		Prerelease: "rc1",
	}, ctx.Semver)
}

func TestSemverOnGitPipeInvalidSemver(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "not-a-semver")
	ctx := testctx.Wrap(t.Context())
	require.ErrorContains(t, Pipe{}.Run(ctx), "failed to parse tag")
}

func TestSemverOnGitPipeInvalidSemverSkipValidate(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "not-a-semver")
	ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Validate))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{}, ctx.Semver)
}

func TestPreviousTagPrerelease(t *testing.T) {
	t.Run("prerelease tag prefers prerelease previous", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
		testlib.GitCommit(t, "commit1")
		testlib.GitTag(t, "v1.0.0")
		testlib.GitCommit(t, "commit2")
		testlib.GitTag(t, "v1.1.0-rc1")
		testlib.GitCommit(t, "commit3")
		testlib.GitTag(t, "v1.1.0-rc2")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "v1.1.0-rc2", ctx.Git.CurrentTag)
		require.Equal(t, "v1.1.0-rc1", ctx.Git.PreviousTag)
		require.Equal(t, "rc2", ctx.Semver.Prerelease)
	})

	t.Run("stable tag skips prerelease previous", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
		testlib.GitCommit(t, "commit1")
		testlib.GitTag(t, "v1.0.0")
		testlib.GitCommit(t, "commit2")
		testlib.GitTag(t, "v1.1.0-rc1")
		testlib.GitCommit(t, "commit3")
		testlib.GitTag(t, "v1.1.0-rc2")
		testlib.GitCommit(t, "commit4")
		testlib.GitTag(t, "v1.1.0")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "v1.1.0", ctx.Git.CurrentTag)
		require.Equal(t, "v1.0.0", ctx.Git.PreviousTag)
		require.Empty(t, ctx.Semver.Prerelease)
	})

	t.Run("stable tag with only prerelease history", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
		testlib.GitCommit(t, "commit1")
		testlib.GitTag(t, "v1.0.0-rc1")
		testlib.GitCommit(t, "commit2")
		testlib.GitTag(t, "v1.0.0")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "v1.0.0", ctx.Git.CurrentTag)
		require.Equal(t, "v1.0.0-rc1", ctx.Git.PreviousTag)
		require.Empty(t, ctx.Semver.Prerelease)
	})

	t.Run("first prerelease has no prerelease previous", func(t *testing.T) {
		testlib.Mktmp(t)
		testlib.GitInit(t)
		testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
		testlib.GitCommit(t, "commit1")
		testlib.GitTag(t, "v1.0.0")
		testlib.GitCommit(t, "commit2")
		testlib.GitTag(t, "v1.1.0-rc1")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, "v1.1.0-rc1", ctx.Git.CurrentTag)
		require.Equal(t, "v1.0.0", ctx.Git.PreviousTag)
		require.Equal(t, "rc1", ctx.Semver.Prerelease)
	})
}
