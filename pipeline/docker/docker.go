// Package docker provides a Pipe that creates and pushes a Docker image
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"

	"github.com/apex/log"

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
	_, err := exec.LookPath("docker")
	if err != nil {
		return ErrNoDocker
	}
	return doRun(ctx)
}

func doRun(ctx *context.Context) error {
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
					var err = process(ctx, folder, docker, binary)
					if err != nil && !pipeline.IsSkip(err) {
						return err
					}
				}
			}
		}
	}
	return nil
}

func process(ctx *context.Context, folder string, docker config.Docker, binary context.Binary) error {
	var root = filepath.Join(ctx.Config.Dist, folder)
	var dockerfile = filepath.Join(root, filepath.Base(docker.Dockerfile))
	var image = fmt.Sprintf("%s:%s", docker.Image, ctx.Version)
	var latest = fmt.Sprintf("%s:latest", docker.Image)

	if err := os.Link(docker.Dockerfile, dockerfile); err != nil {
		return errors.Wrap(err, "failed to link dockerfile")
	}
	for _, file := range docker.Files {
		if err := os.Link(file, filepath.Join(root, filepath.Base(file))); err != nil {
			return errors.Wrapf(err, "failed to link extra file '%s'", file)
		}
	}
	if err := dockerBuild(root, dockerfile, image); err != nil {
		return err
	}
	if docker.Latest {
		if err := dockerTag(image, latest); err != nil {
			return err
		}
	}

	return publish(ctx, docker, image, latest)
}

func publish(ctx *context.Context, docker config.Docker, image, latest string) error {
	// TODO: improve this so it can log it to stdout
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	if err := dockerPush(image); err != nil {
		return err
	}
	ctx.AddDocker(image)
	if !docker.Latest {
		return nil
	}
	if err := dockerTag(image, latest); err != nil {
		return err
	}
	return dockerPush(latest)
}

func dockerBuild(root, dockerfile, image string) error {
	log.WithField("image", image).Info("building docker image")
	var cmd = exec.Command("docker", "build", "-f", dockerfile, "-t", image, root)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to build docker image: \n%s", string(out))
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func dockerTag(image, tag string) error {
	log.WithField("image", image).WithField("tag", tag).Info("tagging docker image")
	var cmd = exec.Command("docker", "tag", image, tag)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to tag docker image: \n%s", string(out))
	}
	log.Debugf("docker tag output: \n%s", string(out))
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
