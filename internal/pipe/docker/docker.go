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
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
)

// ErrNoDocker is shown when docker cannot be found in $PATH
var ErrNoDocker = errors.New("docker not present in $PATH")

// Pipe for docker
type Pipe struct{}

func (Pipe) String() string {
	return "Docker images"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Dockers {
		var docker = &ctx.Config.Dockers[i]

		if docker.Image != "" {
			deprecate.Notice("docker.image")

			if len(docker.TagTemplates) != 0 {
				deprecate.Notice("docker.tag_templates")
			} else {
				docker.TagTemplates = []string{"{{ .Version }}"}
			}
		}

		if docker.Goos == "" {
			docker.Goos = "linux"
		}
		if docker.Goarch == "" {
			docker.Goarch = "amd64"
		}
	}
	// only set defaults if there is exactly 1 docker setup in the config file.
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
	if len(ctx.Config.Dockers) == 0 || missingImage(ctx) {
		return pipe.Skip("docker section is not configured")
	}
	_, err := exec.LookPath("docker")
	if err != nil {
		return ErrNoDocker
	}
	return doRun(ctx)
}

// Publish the docker images
func (Pipe) Publish(ctx *context.Context) error {
	var images = ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableDockerImage)).List()
	for _, image := range images {
		if err := dockerPush(ctx, image); err != nil {
			return err
		}
	}
	return nil
}

func missingImage(ctx *context.Context) bool {
	return ctx.Config.Dockers[0].Image == "" && len(ctx.Config.Dockers[0].ImageTemplates) == 0
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
						return a.ExtraOr("Binary", "").(string) == docker.Binary
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

func process(ctx *context.Context, docker config.Docker, bin artifact.Artifact) error {
	tmp, err := ioutil.TempDir(ctx.Config.Dist, "goreleaserdocker")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary dir")
	}
	log.Debug("tempdir: " + tmp)

	images, err := processImageTemplates(ctx, docker, bin)
	if err != nil {
		return err
	}

	if err := os.Link(docker.Dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return errors.Wrap(err, "failed to link dockerfile")
	}
	for _, file := range docker.Files {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0755); err != nil {
			return errors.Wrapf(err, "failed to link extra file '%s'", file)
		}
		if err := link(file, filepath.Join(tmp, file)); err != nil {
			return errors.Wrapf(err, "failed to link extra file '%s'", file)
		}
	}
	if err := os.Link(bin.Path, filepath.Join(tmp, filepath.Base(bin.Path))); err != nil {
		return errors.Wrap(err, "failed to link binary")
	}

	buildFlags, err := processBuildFlagTemplates(ctx, docker, bin)
	if err != nil {
		return err
	}

	if err := dockerBuild(ctx, tmp, images, buildFlags); err != nil {
		return err
	}
	if docker.SkipPush {
		// TODO: this should also be better handled
		log.Warn(pipe.Skip("skip_push is set").Error())
		return nil
	}
	for _, img := range images {
		ctx.Artifacts.Add(artifact.Artifact{
			Type:   artifact.PublishableDockerImage,
			Name:   img,
			Path:   img,
			Goarch: docker.Goarch,
			Goos:   docker.Goos,
			Goarm:  docker.Goarm,
		})
	}
	return nil
}

func processImageTemplates(ctx *context.Context, docker config.Docker, artifact artifact.Artifact) ([]string, error) {
	if len(docker.ImageTemplates) != 0 && docker.Image != "" {
		return nil, errors.New("failed to process image, use either image_templates (preferred) or image, not both")
	}

	// nolint:prealloc
	var images []string
	for _, imageTemplate := range docker.ImageTemplates {
		// TODO: add overrides support to config
		image, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(imageTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute image template '%s'", imageTemplate)
		}

		images = append(images, image)
	}

	for _, tagTemplate := range docker.TagTemplates {
		imageTemplate := fmt.Sprintf("%s:%s", docker.Image, tagTemplate)
		// TODO: add overrides support to config
		image, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(imageTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to execute tag template '%s'", tagTemplate)
		}
		images = append(images, image)
	}

	return images, nil
}

func processBuildFlagTemplates(ctx *context.Context, docker config.Docker, artifact artifact.Artifact) ([]string, error) {
	// nolint:prealloc
	var buildFlags []string
	for _, buildFlagTemplate := range docker.BuildFlagTemplates {
		buildFlag, err := tmpl.New(ctx).
			WithArtifact(artifact, map[string]string{}).
			Apply(buildFlagTemplate)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process build flag template '%s'", buildFlagTemplate)
		}
		buildFlags = append(buildFlags, buildFlag)
	}
	return buildFlags, nil
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
		}).Debug("extra file")
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		return os.Link(path, dst)
	})
}

func dockerBuild(ctx *context.Context, root string, images, flags []string) error {
	log.WithField("image", images[0]).Info("building docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", buildCommand(images, flags)...)
	cmd.Dir = root
	log.WithField("cmd", cmd.Args).WithField("cwd", cmd.Dir).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to build docker image: \n%s", string(out))
	}
	log.Debugf("docker build output: \n%s", string(out))
	return nil
}

func buildCommand(images, flags []string) []string {
	base := []string{"build", "."}
	for _, image := range images {
		base = append(base, "-t", image)
	}
	base = append(base, flags...)
	return base
}

func dockerPush(ctx *context.Context, image artifact.Artifact) error {
	log.WithField("image", image.Name).Info("pushing docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "push", image.Name)
	log.WithField("cmd", cmd.Args).Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to push docker image: \n%s", string(out))
	}
	log.Debugf("docker push output: \n%s", string(out))
	image.Type = artifact.DockerImage
	ctx.Artifacts.Add(image)
	return nil
}
