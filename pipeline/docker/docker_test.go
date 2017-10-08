package docker

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/apex/log"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"

	"github.com/stretchr/testify/assert"
)

func killAndRm() {
	log.Info("killing registry")
	_ = exec.Command("docker", "kill", "registry").Run()
	_ = exec.Command("docker", "rm", "registry").Run()
}

func TestMain(m *testing.M) {
	killAndRm()
	if err := exec.Command(
		"docker", "run", "-d", "-p", "5000:5000", "--name", "registry", "registry:2",
	).Run(); err != nil {
		log.WithError(err).Fatal("failed to start docker registry")
	}
	defer killAndRm()
	os.Exit(m.Run())
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	_, err = os.Create(binPath)
	assert.NoError(t, err)
	var images = []string{
		"localhost:5000/goreleaser/test_run_pipe:1.0.0",
		"localhost:5000/goreleaser/test_run_pipe:latest",
	}
	// this might fail as the image doesnt exist yet, so lets ignore the error
	for _, img := range images {
		_ = exec.Command("docker", "rmi", img).Run()
	}
	var ctx = &context.Context{
		Version: "1.0.0",
		Publish: true,
		Config: config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			Dockers: []config.Docker{
				{
					Image:      "localhost:5000/goreleaser/test_run_pipe",
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Binary:     "mybin",
					Latest:     true,
				},
				{
					Image:      "localhost:5000/goreleaser/test_run_pipe_nope",
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
	assert.NoError(t, Pipe{}.Run(ctx))

	// this might should not fail as the image should have been created when
	// the step ran
	for _, img := range images {
		assert.NoError(t, exec.Command("docker", "rmi", img).Run())
	}

	// the test_run_pipe_nope image should not have been created, so deleting
	// it should fail
	assert.Error(t,
		exec.Command(
			"docker", "rmi", "localhost:5000/goreleaser/test_run_pipe_nope:1.0.0",
		).Run(),
	)
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestNoDockers(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestNoDockerWithoutImageName(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	}))))
}

func TestDockerNotInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
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
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoDocker.Error())
}
