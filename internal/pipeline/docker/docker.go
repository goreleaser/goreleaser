// Package docker provides a Pipe that creates and pushes a Docker image
package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/errors"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipeline"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
		if len(docker.TagTemplates) == 0 {
			docker.TagTemplates = append(docker.TagTemplates, "{{ .Version }}")
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
	var g = semerrgroup.New(ctx.Parallelism)
	for _, docker := range ctx.Config.Dockers {
		docker := docker
		g.Go(func() error {
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
			if len(binaries) != 1 {
				return fmt.Errorf(
					"%d binaries match docker definition: %s: %s_%s_%s",
					len(binaries),
					docker.Binary, docker.Goos, docker.Goarch, docker.Goarm,
				)
			}
			return process(ctx, docker, binaries[0])
		})
	}
	return g.Wait()
}

func process(ctx *context.Context, docker config.Docker, artifact artifact.Artifact) error {
	tmp, err := ioutil.TempDir(ctx.Config.Dist, "goreleaserdocker")
	if err != nil {
		return errors.Wrap(err, "failed to create temporaty dir")
	}
	defer os.RemoveAll(tmp)
	log.Info("tempdir: " + tmp)
	// nolint:prealloc
	var images []string
	for _, tagTemplate := range docker.TagTemplates {
		// TODO: add overrides support to config
		tag, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(tagTemplate)
		if err != nil {
			return errors.Wrapf(err, "failed to execute tag template '%s'", tagTemplate)
		}
		images = append(images, fmt.Sprintf("%s:%s", docker.Image, tag))
	}
	if err := os.Link(docker.Dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return errors.Wrap(err, "failed to link dockerfile")
	}
	for _, file := range docker.Files {
		if err := link(file, filepath.Join(tmp, filepath.Base(file))); err != nil {
			return errors.Wrapf(err, "failed to link extra file '%s'", file)
		}
	}
	if err := os.Link(artifact.Path, filepath.Join(tmp, filepath.Base(artifact.Path))); err != nil {
		return errors.Wrap(err, "failed to link binary")
	}
	if err := dockerBuild(ctx, tmp, images[0]); err != nil {
		return err
	}
	for _, img := range images[1:] {
		if err := dockerTag(ctx, images[0], img); err != nil {
			return err
		}
	}
	return publish(ctx, docker, images)
}

// walks the src, recreating dirs and hard-linking files
func link(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// We have the following:
		// - src = "a/b"
		// - dest = "dist/linuxamd64/b"
		// - path = "a/b/c.txt"
		// So we join "a/b" with "c.txt" and use it as the destination.
		var dst = filepath.Join(dest, strings.Replace(path, src, "", 1))
		log.WithFields(log.Fields{
			"src": path,
			"dst": dst,
		}).Info("extra file")
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		return os.Link(path, dst)
	})
}

func publish(ctx *context.Context, docker config.Docker, images []string) error {
	if ctx.SkipPublish {
		// TODO: this should be better handled
		log.Warn(pipeline.ErrSkipPublishEnabled.Error())
		return nil
	}
	if docker.SkipPush {
		// TODO: this should also be better handled
		log.Warn(pipeline.Skip("skip_push is set").Error())
		return nil
	}
	for _, image := range images {
		if err := dockerPush(ctx, docker, image); err != nil {
			return err
		}
	}
	return nil
}

func dockerBuild(ctx *context.Context, root, image string) error {
	log.WithField("image", image).Info("building docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "build", "-t", image, ".")
	cmd.Dir = root
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to build docker image: \n%s", string(out))
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func dockerTag(ctx *context.Context, image, tag string) error {
	log.WithField("image", image).WithField("tag", tag).Info("tagging docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "tag", image, tag)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to tag docker image: \n%s", string(out))
	}
	log.Debugf("docker tag output: \n%s", string(out))
	return nil
}

func dockerPush(ctx *context.Context, docker config.Docker, image string) error {
	log.WithField("image", image).Info("pushing docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "push", image)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to push docker image: \n%s", string(out))
	}
	log.Debugf("docker push output: \n%s", string(out))
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.DockerImage,
		Name:   image,
		Path:   image,
		Goarch: docker.Goarch,
		Goos:   docker.Goos,
		Goarm:  docker.Goarm,
	})
	return nil
}
