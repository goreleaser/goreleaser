package git

import (
	"io/ioutil"
	"os"
	"os/exec"
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
	assert.NoError(exec.Command("git", "init").Run())
	assert.NoError(exec.Command("git", "commit", "--allow-empty", "-m", "asd").Run())
	assert.NoError(exec.Command("git", "tag", "v0.0.1").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestNewRepository(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	assert.NoError(exec.Command("git", "init").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}

func TestInvalidTagFormat(t *testing.T) {
	var assert = assert.New(t)
	_, back := createAndChdir(t)
	defer back()
	assert.NoError(exec.Command("git", "init").Run())
	assert.NoError(exec.Command("git", "commit", "--allow-empty", "-m", "asd").Run())
	assert.NoError(exec.Command("git", "tag", "sadasd").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(Pipe{}.Run(ctx), "sadasd is not in a valid version format")
}

func createAndChdir(t *testing.T) (current string, back func()) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	previous, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	return folder, func() {
		assert.NoError(os.Chdir(previous))
	}
}
