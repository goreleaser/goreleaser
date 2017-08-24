package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestNotAGitFolder(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestSingleCommit(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal("v0.0.1", ctx.Git.CurrentTag)
}

func TestNewRepository(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestNoTagsSnapshot(t *testing.T) {
	assert := assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{
				NameTemplate: defaults.SnapshotNameTemplate,
			},
		},
		Snapshot: true,
		Publish:  false,
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Contains(ctx.Version, "SNAPSHOT-")
}

func TestNoTagsSnapshotInvalidTemplate(t *testing.T) {
	assert := assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{
				NameTemplate: "{{",
			},
		},
		Snapshot: true,
		Publish:  false,
	}
	assert.Error(Pipe{}.Run(ctx))
}

// TestNoTagsNoSnapshot covers the situation where a repository
// only contains simple commits and no tags. In this case you have
// to set the --snapshot flag otherwise an error is returned.
func TestNoTagsNoSnapshot(t *testing.T) {
	assert := assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = &context.Context{
		Config: config.Project{
			Snapshot: config.Snapshot{
				NameTemplate: defaults.SnapshotNameTemplate,
			},
		},
		Snapshot: false,
		Publish:  false,
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestInvalidTagFormat(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "sadasd")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	assert.EqualError(Pipe{}.Run(ctx), "sadasd is not in a valid version format")
	assert.Equal("sadasd", ctx.Git.CurrentTag)
}

func TestDirty(t *testing.T) {
	var assert = assert.New(t)
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	assert.NoError(err)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(ioutil.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0644))
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	err = Pipe{}.Run(ctx)
	assert.Error(err)
	assert.Contains(err.Error(), "git is currently in a dirty state:")
}

func TestTagIsNotLastCommit(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	err := Pipe{}.Run(ctx)
	assert.Error(err)
	assert.Contains(err.Error(), "git tag v0.0.1 was not made against commit")
}

func TestValidState(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.2")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Equal("v0.0.2", ctx.Git.CurrentTag)
	assert.NotContains("commit4", ctx.ReleaseNotes)
	assert.NotContains("commit3", ctx.ReleaseNotes)
}

func TestNoValidate(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit5")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit6")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: false,
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestChangelog(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "added feature 1")
	testlib.GitCommit(t, "fixed bug 2")
	testlib.GitTag(t, "v0.0.2")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal("v0.0.2", ctx.Git.CurrentTag)
	assert.Contains(ctx.ReleaseNotes, "## Changelog")
	assert.NotContains(ctx.ReleaseNotes, "first")
	assert.Contains(ctx.ReleaseNotes, "added feature 1")
	assert.Contains(ctx.ReleaseNotes, "fixed bug 2")
}

func TestChangelogOfFirstRelease(t *testing.T) {
	var assert = assert.New(t)
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
	var ctx = &context.Context{
		Config: config.Project{},
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal("v0.0.1", ctx.Git.CurrentTag)
	assert.Contains(ctx.ReleaseNotes, "## Changelog")
	for _, msg := range msgs {
		assert.Contains(ctx.ReleaseNotes, msg)
	}
}

func TestCustomReleaseNotes(t *testing.T) {
	var assert = assert.New(t)
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config:       config.Project{},
		ReleaseNotes: "custom",
	}
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal("v0.0.1", ctx.Git.CurrentTag)
	assert.Equal(ctx.ReleaseNotes, "custom")
}
