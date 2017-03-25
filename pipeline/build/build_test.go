package build

import (
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRun(t *testing.T) {
	assert.NoError(t, run(runtime.GOOS, runtime.GOARCH, []string{"go", "list", "./..."}))
}

func TestRunInvalidCommand(t *testing.T) {
	assert.Error(t, run(runtime.GOOS, runtime.GOARCH, []string{"gggggo", "nope"}))
}

func TestBuild(t *testing.T) {
	assert := assert.New(t)
	var config = config.Project{
		Build: config.Build{
			Binary: "testing",
			Flags:  "-n",
		},
	}
	var ctx = &context.Context{
		Config: config,
	}
	assert.NoError(build("build_test", runtime.GOOS, runtime.GOARCH, ctx))
}
