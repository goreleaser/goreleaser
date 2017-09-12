package docker

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"

	"github.com/stretchr/testify/assert"
)

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	assert.NoError(os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	_, err = os.Create(binPath)
	assert.NoError(err)
	// this might fail as the image doesnt exist yet, so lets ignore the error
	_ = exec.Command("docker", "rmi", "goreleaser/test_run_pipe:v1.0.0").Run()
	var ctx = &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.0.0",
		},
		Publish: true,
		Config: config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			Dockers: []config.Docker{
				{
					Image:      "goreleaser/test_run_pipe",
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Binary:     "mybin",
				},
				{
					Image:      "goreleaser/test_run_pipe_nope",
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Binary:     "otherbin",
				},
			},
		},
	}
	for _, plat := range []string{"linuxamd64", "linux386", "darwinamd64"} {
		ctx.AddBinary(plat, "mybin", "mybin", binPath)
	}
	assert.NoError(Pipe{}.Run(ctx))
	// this might should not fail as the image should have been created when
	// the step ran
	assert.NoError(
		exec.Command("docker", "rmi", "goreleaser/test_run_pipe:v1.0.0").Run(),
	)
	// the test_run_pipe_nope image should not have been created, so deleting
	// it should fail
	assert.Error(
		exec.Command("docker", "rmi", "goreleaser/test_run_pipe_nope:v1.0.0").Run(),
	)
}

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
