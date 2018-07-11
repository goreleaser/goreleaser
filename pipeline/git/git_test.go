package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNotAGitFolder(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNotRepository.Error())
}

func TestSingleCommit(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
}

func TestNewRepository(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	// TODO: improve this error handling
	assert.Contains(t, Pipe{}.Run(ctx).Error(), `fatal: ambiguous argument 'HEAD'`)
}

func TestNoTagsSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "SNAPSHOT-{{.Commit}}",
		},
	})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.Version, "SNAPSHOT-")
}

func TestNoTagsSnapshotInvalidTemplate(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{",
		},
	})
	ctx.Snapshot = true
	assert.EqualError(t, Pipe{}.Run(ctx), `failed to generate snapshot name: template: tmpl:1: unexpected unclosed action in command`)
}

// TestNoTagsNoSnapshot covers the situation where a repository
// only contains simple commits and no tags. In this case you have
// to set the --snapshot flag otherwise an error is returned.
func TestNoTagsNoSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	ctx.Snapshot = false
	assert.EqualError(t, Pipe{}.Run(ctx), `git doesn't contain any tags. Either add a tag or use --snapshot`)
}

func TestInvalidTagFormat(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "sadasd")
	var ctx = context.New(config.Project{})
	assert.EqualError(t, Pipe{}.Run(ctx), "sadasd is not in a valid version format")
	assert.Equal(t, "sadasd", ctx.Git.CurrentTag)
}

func TestDirty(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	assert.NoError(t, err)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(t, ioutil.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0644))
	t.Run("all checks up", func(t *testing.T) {
		err = Pipe{}.Run(context.New(config.Project{}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git is currently in a dirty state:")
	})
	t.Run("skip validate is set", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipValidate = true
		err = Pipe{}.Run(ctx)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("snapshot", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.Snapshot = true
		err = Pipe{}.Run(ctx)
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
}

func TestTagIsNotLastCommit(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	err := Pipe{}.Run(context.New(config.Project{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git tag v0.0.1 was not made against commit")
}

func TestValidState(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{})
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
}

func TestSnapshotNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal(t, fakeInfo.CurrentTag, ctx.Git.CurrentTag)
}

func TestSnapshotNoCommits(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotWithoutRepo(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Equal(t, fakeInfo, ctx.Git)
}

func TestSnapshotDirty(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(t, ioutil.WriteFile(filepath.Join(folder, "foo"), []byte("foobar"), 0644))
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestShortCommitHash(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{
		Snapshot: config.Snapshot{
			NameTemplate: "{{.Commit}}",
		},
	})
	ctx.Snapshot = true
	ctx.Config.Git.ShortHash = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Version, 7)
}
