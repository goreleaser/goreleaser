package git

import (
	"io/ioutil"
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
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.doRun(ctx, folder))
}

func TestSingleCommit(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	assert.NoError(exec.Command("git", "-C", folder, "init").Run())
	assert.NoError(exec.Command("git", "-C", folder, "commit", "--allow-empty", "-m", "asd").Run())
	assert.NoError(exec.Command("git", "-C", folder, "tag", "v0.0.1").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.doRun(ctx, folder))
}

func TestNewRepository(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	assert.NoError(exec.Command("git", "-C", folder, "init").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.doRun(ctx, folder))
}

func TestInvalidTagFormat(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	assert.NoError(exec.Command("git", "-C", folder, "init").Run())
	assert.NoError(exec.Command("git", "-C", folder, "commit", "--allow-empty", "-m", "asd").Run())
	assert.NoError(exec.Command("git", "-C", folder, "tag", "sadasd").Run())
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.EqualError(Pipe{}.doRun(ctx, folder), "sadasd is not in a valid version format")
}
