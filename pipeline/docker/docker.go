// Package docker provides a Pipe that creates and pushes a Docker image
package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
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
	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, docker := range ctx.Config.Dockers {
		docker := docker
		sem <- true
		g.Go(func() error {
			defer func() {
				<-sem
			}()
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
				log.Warnf("no binaries found for %s", docker.Binary)
			}
			for _, binary := range binaries {
				if err := process(ctx, docker, binary); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
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
	if err := dockerBuild(ctx, root, dockerfile, image); err != nil {
		return err
	}
	if docker.Latest {
		if err := dockerTag(ctx, image, latest); err != nil {
			return err
		}
	}

	return publish(ctx, docker, image, latest)
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

func publish(ctx *context.Context, docker config.Docker, image, latest string) error {
	if !ctx.Publish {
		log.Warn("skipping push because --skip-publish is set")
		return nil
	}
	if err := dockerPush(ctx, docker, image); err != nil {
		return err
	}
	if !docker.Latest {
		return nil
	}
	if err := dockerTag(ctx, image, latest); err != nil {
		return err
	}
	return dockerPush(ctx, docker, latest)
}

func dockerBuild(ctx *context.Context, root, dockerfile, image string) error {
	log.WithField("image", image).Info("building docker image")
	/* #nosec */
	var cmd = exec.CommandContext(ctx, "docker", "build", "-f", dockerfile, "-t", image, root)
	log.WithField("cmd", cmd).Debug("executing")
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
	log.WithField("cmd", cmd).Debug("executing")
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
	log.WithField("cmd", cmd).Debug("executing")
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
