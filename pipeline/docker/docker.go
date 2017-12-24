// Package docker provides a Pipe that creates and pushes a Docker image
package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/apex/log"
	"github.com/pkg/errors"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
	"io/ioutil"
)

// ErrNoDocker is shown when docker cannot be found in $PATH
var ErrNoDocker = errors.New("docker not present in $PATH")

// Pipe for docker
type Pipe struct{}

func (Pipe) String() string {
	return "creating Docker images"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Dockers {
		var docker = &ctx.Config.Dockers[i]
		if docker.TagTemplate == "" {
			docker.TagTemplate = "{{ .Version }}"
		}
		if docker.Goos == "" {
			docker.Goos = "linux"
		}
		if docker.Goarch == "" {
			docker.Goarch = "amd64"
		}
	}
	// only set defaults if there is exacly 1 docker setup in the config file.
	if len(ctx.Config.Dockers) != 1 {
		return nil
	}
	if ctx.Config.Dockers[0].Binary == "" {
		ctx.Config.Dockers[0].Binary = ctx.Config.Builds[0].Binary
	}
	if ctx.Config.Dockers[0].Dockerfile == "" {
		ctx.Config.Dockers[0].Dockerfile = "Dockerfile"
	}
	return nil
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
	// TODO: could be done in parallel.
	for _, docker := range ctx.Config.Dockers {
		log.WithField("docker", docker).Debug("looking for binaries matching")
		var binaries = ctx.Artifacts.Filter(
			artifact.And(
				artifact.ByGoos(docker.Goos),
				artifact.ByGoarch(docker.Goarch),
				artifact.ByGoarm(docker.Goarm),
				artifact.ByType(artifact.Binary),
				func(a artifact.Artifact) bool {
					return a.Extra["Binary"] == docker.Binary
				},
			),
		).List()
		if len(binaries) == 0 {
			log.Warn("no binaries found")
		}
		for _, binary := range binaries {
			var err = process(ctx, docker, binary)
			if err != nil && !pipeline.IsSkip(err) {
				return err
			}
		}
	}
	return nil
}

func tagName(ctx *context.Context, docker config.Docker) (string, error) {
	var out bytes.Buffer
	t, err := template.New("tag").Parse(docker.TagTemplate)
	if err != nil {
		return "", err
	}
	data := struct {
		Version, Tag string
		Env          map[string]string
	}{
		Version: ctx.Version,
		Tag:     ctx.Git.CurrentTag,
		Env:     ctx.Env,
	}
	err = t.Execute(&out, data)
	return out.String(), err
}

func process(ctx *context.Context, docker config.Docker, artifact artifact.Artifact) error {
	var root = filepath.Dir(artifact.Path)
	var dockerfile = filepath.Join(root, filepath.Base(docker.Dockerfile))
	tag, err := tagName(ctx, docker)
	if err != nil {
		return err
	}
	var image = fmt.Sprintf("%s:%s", docker.Image, tag)
	var latest = fmt.Sprintf("%s:latest", docker.Image)

	if err := os.Link(docker.Dockerfile, dockerfile); err != nil {
		return errors.Wrap(err, "failed to link dockerfile")
	}
	for _, file := range docker.Files {
		if err := link(file, filepath.Join(root, filepath.Base(file))); err != nil {
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

// link a file or directory hard
func link(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return directoryLink(src, dest, info)
	}
	return fileLink(src, dest)
}

// directoryLink recursively creates all subdirectories and links all files hard
func directoryLink(src, dest string, info os.FileInfo) error {
	if info == nil {
		i, err := os.Stat(src)
		if err != nil {
			return err
		}
		info = i
	}
	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}
	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if info.IsDir() {
			err := directoryLink(
				filepath.Join(src, info.Name()),
				filepath.Join(dest, info.Name()),
				info,
			)
			if err != nil {
				return err
			}
		} else {
			err := fileLink(
				filepath.Join(src, info.Name()),
				filepath.Join(dest, info.Name()),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// fileLink links a file hard
func fileLink(src, dest string) error {
	return os.Link(src, dest)
}

func publish(ctx *context.Context, docker config.Docker, image, latest string) error {
	// TODO: improve this so it can log it to stdout
	if !ctx.Publish {
		return pipeline.Skip("--skip-publish is set")
	}
	if ctx.Config.Release.Draft {
		return pipeline.Skip("release is marked as draft")
	}
	if err := dockerPush(ctx, image); err != nil {
		return err
	}
	if !docker.Latest {
		return nil
	}
	if err := dockerTag(image, latest); err != nil {
		return err
	}
	return dockerPush(ctx, latest)
}

func dockerBuild(root, dockerfile, image string) error {
	log.WithField("image", image).Info("building docker image")
	/* #nosec */
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
	/* #nosec */
	var cmd = exec.Command("docker", "tag", image, tag)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to tag docker image: \n%s", string(out))
	}
	log.Debugf("docker tag output: \n%s", string(out))
	return nil
}

func dockerPush(ctx *context.Context, image string) error {
	log.WithField("image", image).Info("pushing docker image")
	/* #nosec */
	var cmd = exec.Command("docker", "push", image)
	log.WithField("cmd", cmd).Debug("executing")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to push docker image: \n%s", string(out))
	}
	log.Debugf("docker push output: \n%s", string(out))
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.DockerImage,
		Name: image,
		// TODO: are the rest of the params relevant here?
	})
	return nil
}
