package docker

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/tj/assert"
)

func TestDescription(t *testing.T) {
	var assert = assert.New(t)
	assert.NotEmpty(Pipe{}.Description())
}

func TestNoDockers(t *testing.T) {
	var assert = assert.New(t)
	assert.True(pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestNoDockerWithoutImageName(t *testing.T) {
	var assert = assert.New(t)
	assert.True(pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	}))))
}

func TestDockerNotInPath(t *testing.T) {
	var assert = assert.New(t)
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(os.Setenv("PATH", path))
	}()
	assert.NoError(os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Version: "1.0.0",
		Config: config.Project{
			Dockers: []config.Docker{
				{
					Image: "a/b",
				},
			},
		},
	}
	assert.EqualError(Pipe{}.Run(ctx), ErrNoDocker.Error())
}
