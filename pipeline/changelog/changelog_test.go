package changelog

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
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
	assert.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
	assert.Contains(t, ctx.ReleaseNotes, "## Changelog")
	assert.NotContains(t, ctx.ReleaseNotes, "first")
	assert.Contains(t, ctx.ReleaseNotes, "added feature 1")
	assert.Contains(t, ctx.ReleaseNotes, "fixed bug 2")
	assert.NotContains(t, ctx.ReleaseNotes, "docs")
	assert.NotContains(t, ctx.ReleaseNotes, "ignored")
	assert.NotContains(t, ctx.ReleaseNotes, "cArs")
	assert.NotContains(t, ctx.ReleaseNotes, "from goreleaser/some-branch")
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

func TestChangelogNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	assert.Error(t, Pipe{}.Run(ctx))
	assert.Empty(t, ctx.ReleaseNotes)
}
