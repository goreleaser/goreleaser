package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestValidVersion(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.NotEmpty(ctx.Git.CurrentTag)
	assert.NotEmpty(ctx.Git.PreviousTag)
	assert.NotEmpty(ctx.Git.Diff)
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
	gitLog(t)
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
	gitLog(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(Pipe{}.Run(ctx), "sadasd is not in a valid version format")
	assert.Equal("sadasd", ctx.Git.CurrentTag)
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
	out, err := git("commit", "--allow-empty", "-m", msg)
	assert.NoError(err)
	assert.Contains(out, "master", msg)
}

func gitTag(t *testing.T, tag string) {
	var assert = assert.New(t)
	out, err := git("tag", tag)
	assert.NoError(err)
	assert.Empty(out)
}

func gitLog(t *testing.T) {
	var assert = assert.New(t)
	out, err := git("log")
	assert.NoError(err)
	assert.NotEmpty(out)
	fmt.Print("\n\ngit log output:\n", out)
}
