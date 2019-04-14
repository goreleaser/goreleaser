package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.1", ctx.Git.CurrentTag)
}

func TestNoRemote(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitCommit(t, "commit1")
	testlib.GitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), "couldn't get remote URL: fatal: No remote configured to list refs from.")
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

// TestNoTagsNoSnapshot covers the situation where a repository
// only contains simple commits and no tags. In this case you have
// to set the --snapshot flag otherwise an error is returned.
func TestNoTagsNoSnapshot(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "first")
	var ctx = context.New(config.Project{})
	ctx.Snapshot = false
	assert.EqualError(t, Pipe{}.Run(ctx), `git doesn't contain any tags. Either add a tag or use --snapshot`)
}

func TestDirty(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	assert.NoError(t, err)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "commit2")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(t, ioutil.WriteFile(dummy.Name(), []byte("lorem ipsum"), 0644))
	t.Run("all checks up", func(t *testing.T) {
		err = Pipe{}.Run(context.New(config.Project{}))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git is currently in a dirty state")
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
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
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
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitCommit(t, "commit3")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "commit4")
	testlib.GitTag(t, "v0.0.2")
	var ctx = context.New(config.Project{})
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "v0.0.2", ctx.Git.CurrentTag)
	assert.Equal(t, "git@github.com:foo/bar.git", ctx.Git.URL)
}

func TestSnapshotNoTags(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
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
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
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
	testlib.GitRemoteAdd(t, "git@github.com:foo/bar.git")
	testlib.GitAdd(t)
	testlib.GitCommit(t, "whatever")
	testlib.GitTag(t, "v0.0.1")
	assert.NoError(t, ioutil.WriteFile(filepath.Join(folder, "foo"), []byte("foobar"), 0644))
	var ctx = context.New(config.Project{})
	ctx.Snapshot = true
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestGitNotInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	assert.EqualError(t, Pipe{}.Run(context.New(config.Project{})), ErrNoGit.Error())
}
