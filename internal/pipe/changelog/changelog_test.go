package changelog

import (
	"io/ioutil"
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

func TestChangelogProvidedViaFlag(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "testdata/changes.md"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffeee\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagAndSkipEnabled(t *testing.T) {
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Skip: true,
		},
	})
	ctx.ReleaseNotes = "testdata/changes.md"
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "c0ff33 coffeee\n", ctx.ReleaseNotes)
}

func TestChangelogProvidedViaFlagDoesntExist(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "testdata/changes.nope"
	require.EqualError(t, Pipe{}.Run(ctx), "open testdata/changes.nope: no such file or directory")
}

func TestChangelogSkip(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Config.Changelog.Skip = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestSnapshot(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestChangelog(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v0.0.2"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1")
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2")
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	bts, err := ioutil.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogForGitlab(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitCommit(t, "ignored: whatever")
	testlib.GitCommit(t, "docs: whatever")
	testlib.GitCommit(t, "something about cArs we dont need")
	testlib.GitCommit(t, "feat: added that thing")
	testlib.GitCommit(t, "Merge pull request #999 from goreleaser/some-branch")
	testlib.GitCommit(t, "this is not a Merge pull request")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{
		Dist: folder,
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"docs:",
					"ignored:",
					"(?i)cars",
					"^Merge pull request",
				},
			},
		},
	})
	ctx.TokenType = context.TokenTypeGitLab
	ctx.Git.CurrentTag = "v0.0.2"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	require.NotContains(t, ctx.ReleaseNotes, "first")
	require.Contains(t, ctx.ReleaseNotes, "added feature 1") // no whitespace because its the last entry of the changelog
	require.Contains(t, ctx.ReleaseNotes, "fixed bug 2   ")  // whitespaces are on purpose
	require.NotContains(t, ctx.ReleaseNotes, "docs")
	require.NotContains(t, ctx.ReleaseNotes, "ignored")
	require.NotContains(t, ctx.ReleaseNotes, "cArs")
	require.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")

	bts, err := ioutil.ReadFile(filepath.Join(folder, "CHANGELOG.md"))
	require.NoError(t, err)
	require.NotEmpty(t, string(bts))
}

func TestChangelogSort(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.9.9")
	testlib.GitCommit(t, "c: commit")
	testlib.GitCommit(t, "a: commit")
	testlib.GitCommit(t, "b: commit")
	testlib.GitTag(t, "v1.0.0")
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{},
	})
	ctx.Git.CurrentTag = "v1.0.0"

	for _, cfg := range []struct {
		Sort    string
		Entries []string
	}{
		{
			Sort: "",
			Entries: []string{
				"b: commit",
				"a: commit",
				"c: commit",
			},
		},
		{
			Sort: "asc",
			Entries: []string{
				"a: commit",
				"b: commit",
				"c: commit",
			},
		},
		{
			Sort: "desc",
			Entries: []string{
				"c: commit",
				"b: commit",
				"a: commit",
			},
		},
	} {
		t.Run("changelog sort='"+cfg.Sort+"'", func(t *testing.T) {
			ctx.Config.Changelog.Sort = cfg.Sort
			entries, err := buildChangelog(ctx)
			require.NoError(t, err)
			require.Len(t, entries, len(cfg.Entries))
			var changes []string
			for _, line := range entries {
				_, msg := extractCommitInfo(line)
				changes = append(changes, msg)
			}
			require.EqualValues(t, cfg.Entries, changes)
		})
	}
}

func TestChangelogInvalidSort(t *testing.T) {
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Sort: "dope",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), ErrInvalidSortDirection.Error())
}

func TestChangelogOfFirstRelease(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}

func TestChangelogFilterInvalidRegex(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commitssss")
	testlib.GitTag(t, "v0.0.3")
	testlib.GitCommit(t, "commitzzz")
	testlib.GitTag(t, "v0.0.4")
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Filters: config.Filters{
				Exclude: []string{
					"(?iasdr4qasd)not a valid regex i guess",
				},
			},
		},
	})
	ctx.Git.CurrentTag = "v0.0.4"
	require.EqualError(t, Pipe{}.Run(ctx), "error parsing regexp: invalid or unsupported Perl syntax: `(?ia`")
}

func TestChangelogNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	require.Error(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ReleaseNotes)
}

func TestChangelogOnBranchWithSameNameAsTag(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var msgs = []string{
		"initial commit",
		"another one",
		"one more",
		"and finally this one",
	}
	for _, msg := range msgs {
		testlib.GitCommit(t, msg)
	}
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCheckoutBranch(t, "v0.0.1")
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "v0.0.1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		require.Contains(t, ctx.ReleaseNotes, msg)
	}
}
