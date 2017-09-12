// Package docker provides a Pipe that creates and pushes a Docker image
package docker

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/pkg/errors"
)

// ErrNoDocker is shown when docker cannot be found in $PATH
var ErrNoDocker = errors.New("docker not present in $PATH")

// Pipe for docker
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating Docker images"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if len(ctx.Config.Dockers) == 0 || ctx.Config.Dockers[0].Image == "" {
		return pipeline.Skip("docker section is not configured")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	_, err := exec.LookPath("docker")
	if err != nil {
		return ErrNoDocker
	}
	for _, docker := range ctx.Config.Dockers {
		var imagePlatform = docker.Goos + docker.Goarch + docker.Goarm
		for platform, groups := range ctx.Binaries {
			if platform != imagePlatform {
				continue
			}
			for folder, binaries := range groups {
				for _, binary := range binaries {
					if binary.Name != docker.Binary {
						continue
					}
					var err = doRun(ctx, folder, docker, binary)
					if err != nil && !pipeline.IsSkip(err) {
						return err
					}
				}
			}
		}
	}
	return nil
}

func doRun(ctx *context.Context, folder string, docker config.Docker, binary context.Binary) error {
	var root = filepath.Join(ctx.Config.Dist, folder)
	var dockerfile = filepath.Join(root, "Dockerfile")
	var image = fmt.Sprintf("%s:%s", docker.Image, ctx.Git.CurrentTag)

	bts, err := ioutil.ReadFile(docker.Dockerfile)
	if err != nil {
		return errors.Wrap(err, "failed to read dockerfile")
	}
	if err := ioutil.WriteFile(dockerfile, bts, 0755); err != nil {
		return err
	}
	log.WithField("file", dockerfile).Debug("wrote dockerfile")
	if err := dockerBuild(root, image); err != nil {
		return err
	}
	// TODO: improve this so it can log into to stdout
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if err := dockerPush(image); err != nil {
		return err
	}
	return nil
}

func dockerBuild(root, image string) error {
	log.WithField("image", image).Info("building docker image")
	var cmd = exec.Command("docker", "build", "-t", image, root)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to build docker image: \n%s", string(out))
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func dockerPush(image string) error {
	log.WithField("image", image).Info("pushing docker image")
	var cmd = exec.Command("docker", "push", image)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to push docker image: \n%s", string(out))
	}
	log.Debugf("docker push output: \n%s", string(out))
	return nil
}
