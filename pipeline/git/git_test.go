package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestNotAGitFolder(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestSingleCommit(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "commit1")
	gitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Equal("v0.0.1", ctx.Git.CurrentTag)
}

func TestNewRepository(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestInvalidTagFormat(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "commit2")
	gitTag(t, "sadasd")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	assert.EqualError(Pipe{}.Run(ctx), "sadasd is not in a valid version format")
	assert.Equal("sadasd", ctx.Git.CurrentTag)
}

func TestDirty(t *testing.T) {
	var assert = assert.New(t)
	folder, back := createAndChdir(t)
	defer back()
	gitInit(t)
	dummy, err := os.Create(filepath.Join(folder, "dummy"))
	assert.NoError(err)
	gitAdd(t)
	gitCommit(t, "commit2")
	gitTag(t, "v0.0.1")
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
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "commit3")
	gitTag(t, "v0.0.1")
	gitCommit(t, "commit4")
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
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "commit3")
	gitTag(t, "v0.0.1")
	gitCommit(t, "commit4")
	gitTag(t, "v0.0.2")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: true,
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestNoValidate(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitAdd(t)
	gitCommit(t, "commit5")
	gitTag(t, "v0.0.1")
	gitCommit(t, "commit6")
	var ctx = &context.Context{
		Config:   config.Project{},
		Validate: false,
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestChangelog(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "first")
	gitTag(t, "v0.0.1")
	gitCommit(t, "added feature 1")
	gitCommit(t, "fixed bug 2")
	gitTag(t, "v0.0.2")
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Equal("v0.0.2", ctx.Git.CurrentTag)
	assert.Contains(ctx.ReleaseNotes, "## Changelog")
	assert.Contains(ctx.ReleaseNotes, "added feature 1")
	assert.Contains(ctx.ReleaseNotes, "fixed bug 2")
}

func TestCustomReleaseNotes(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	gitInit(t)
	gitCommit(t, "first")
	gitTag(t, "v0.0.1")
	var ctx = &context.Context{
		Config:       config.Project{},
		ReleaseNotes: "custom",
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Equal("v0.0.1", ctx.Git.CurrentTag)
	assert.Equal(ctx.ReleaseNotes, "custom")
}

//
// helper functions
//

func createAndChdir(t *testing.T) (current string, back func()) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	previous, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	return folder, func() {
		assert.NoError(os.Chdir(previous))
	}
}

func gitInit(t *testing.T) {
	var assert = assert.New(t)
	out, err := git("init")
	assert.NoError(err)
	assert.Contains(out, "Initialized empty Git repository")
}

func gitCommit(t *testing.T, msg string) {
	var assert = assert.New(t)
	out, err := fakeGit("commit", "--allow-empty", "-m", msg)
	assert.NoError(err)
	assert.Contains(out, "master", msg)
}

func gitTag(t *testing.T, tag string) {
	var assert = assert.New(t)
	out, err := fakeGit("tag", tag)
	assert.NoError(err)
	assert.Empty(out)
}

func gitAdd(t *testing.T) {
	var assert = assert.New(t)
	out, err := git("add", "-A")
	assert.NoError(err)
	assert.Empty(out)
}

func fakeGit(args ...string) (string, error) {
	var allArgs = []string{
		"-c",
		"user.name='GoReleaser'",
		"-c",
		"user.email='test@goreleaser.github.com'",
	}
	allArgs = append(allArgs, args...)
	return git(allArgs...)
}
