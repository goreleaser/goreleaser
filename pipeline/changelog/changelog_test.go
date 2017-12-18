package changelog

import (
	"testing"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestChangelogProvidedViaFlag(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.ReleaseNotes = "c0ff33 foo bar"
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestSnapshot(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestChangelog(t *testing.T) {
	_, back := testlib.Mktmp(t)
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
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.ReleaseNotes, "## Changelog")
	assert.NotContains(t, ctx.ReleaseNotes, "first")
	assert.Contains(t, ctx.ReleaseNotes, "added feature 1")
	assert.Contains(t, ctx.ReleaseNotes, "fixed bug 2")
	assert.NotContains(t, ctx.ReleaseNotes, "docs")
	assert.NotContains(t, ctx.ReleaseNotes, "ignored")
	assert.NotContains(t, ctx.ReleaseNotes, "cArs")
	assert.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")
}

func TestChangelogSort(t *testing.T) {
	f, back := testlib.Mktmp(t)
	log.Info(f)
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
			assert.NoError(t, err)
			assert.Len(t, entries, len(cfg.Entries))
			var changes []string
			for _, line := range entries {
				_, msg := extractCommitInfo(line)
				changes = append(changes, msg)
			}
			assert.EqualValues(t, cfg.Entries, changes)
		})
	}
}

func TestChangelogInvalidSort(t *testing.T) {
	var ctx = context.New(config.Project{
		Changelog: config.Changelog{
			Sort: "dope",
		},
	})
	assert.EqualError(t, Pipe{}.Run(ctx), ErrInvalidSortDirection.Error())
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
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		assert.Contains(t, ctx.ReleaseNotes, msg)
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
	assert.EqualError(t, Pipe{}.Run(ctx), "error parsing regexp: invalid or unsupported Perl syntax: `(?ia`")
}

func TestChangelogNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	assert.Error(t, Pipe{}.Run(ctx))
	assert.Empty(t, ctx.ReleaseNotes)
}
