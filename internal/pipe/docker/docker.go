package docker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	dockerConfigExtra = "DockerConfig"

	useBuildx = "buildx"
	useDocker = "docker"
)

// Pipe for docker.
type Pipe struct{}

func (Pipe) String() string                 { return "docker images" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Dockers) == 0 || ctx.SkipDocker }

func (Pipe) Dependencies(ctx *context.Context) []string {
	var cmds []string
	for _, s := range ctx.Config.Dockers {
		switch s.Use {
		case useDocker, useBuildx:
			cmds = append(cmds, "docker")
			// TODO: how to check if buildx is installed
		}
	}
	return cmds
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("dockers")
	for i := range ctx.Config.Dockers {
		docker := &ctx.Config.Dockers[i]

		if docker.ID != "" {
			ids.Inc(docker.ID)
		}
		if docker.Goos == "" {
			docker.Goos = "linux"
		}
		if docker.Goarch == "" {
			docker.Goarch = "amd64"
		}
		if docker.Goarm == "" {
			docker.Goarm = "6"
		}
		if docker.Goamd64 == "" {
			docker.Goamd64 = "v1"
		}
		if docker.Dockerfile == "" {
			docker.Dockerfile = "Dockerfile"
		}
		if docker.Use == "" {
			docker.Use = useDocker
		}
		if err := validateImager(docker.Use); err != nil {
			return err
		}
	}
	return ids.Validate()
}

func validateImager(use string) error {
	valid := make([]string, 0, len(imagers))
	for k := range imagers {
		valid = append(valid, k)
	}
	for _, s := range valid {
		if s == use {
			return nil
		}
	}
	sort.Strings(valid)
	return fmt.Errorf("docker: invalid use: %s, valid options are %v", use, valid)
}

// Publish the docker images.
func (Pipe) Publish(ctx *context.Context) error {
	skips := pipe.SkipMemento{}
	images := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableDockerImage)).List()
	for _, image := range images {
		if err := dockerPush(ctx, image); err != nil {
			if pipe.IsSkip(err) {
				skips.Remember(err)
				continue
			}
			return err
		}
	}
	return skips.Evaluate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for i, docker := range ctx.Config.Dockers {
		i := i
		docker := docker
		g.Go(func() error {
			log := log.WithField("index", i)
			log.Debug("looking for artifacts matching")
			filters := []artifact.Filter{
				artifact.ByGoos(docker.Goos),
				artifact.ByGoarch(docker.Goarch),
				artifact.Or(
					artifact.ByType(artifact.Binary),
					artifact.ByType(artifact.LinuxPackage),
				),
			}
			// TODO: properly test this
			switch docker.Goarch {
			case "amd64":
				filters = append(filters, artifact.ByGoamd64(docker.Goamd64))
			case "arm":
				filters = append(filters, artifact.ByGoarm(docker.Goarm))
			}
			if len(docker.IDs) > 0 {
				filters = append(filters, artifact.ByIDs(docker.IDs...))
			}
			artifacts := ctx.Artifacts.Filter(artifact.And(filters...))
			log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")
			return process(ctx, docker, artifacts.List())
		})
	}
	if err := g.Wait(); err != nil {
		if pipe.IsSkip(err) {
			return err
		}
		return fmt.Errorf("docker build failed: %w\nLearn more at https://goreleaser.com/errors/docker-build\n", err) // nolint:revive
	}
	return nil
}

func process(ctx *context.Context, docker config.Docker, artifacts []*artifact.Artifact) error {
	if len(artifacts) == 0 {
		log.Warn("not binaries or packages found for the given platform - COPY/ADD may not work")
	}
	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}

	images, err := processImageTemplates(ctx, docker)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return pipe.Skip("no image templates found")
	}

	log := log.WithField("image", images[0])
	log.Debug("tempdir: " + tmp)

	dockerfile, err := tmpl.New(ctx).Apply(docker.Dockerfile)
	if err != nil {
		return err
	}
	if err := gio.Copy(dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
		return fmt.Errorf("failed to copy dockerfile: %w", err)
	}

	for _, file := range docker.Files {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0o755); err != nil {
			return fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
		if err := gio.Copy(file, filepath.Join(tmp, file)); err != nil {
			return fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
	}
	for _, art := range artifacts {
		if err := gio.Copy(art.Path, filepath.Join(tmp, filepath.Base(art.Path))); err != nil {
			return fmt.Errorf("failed to copy artifact: %w", err)
		}
	}

	buildFlags, err := processBuildFlagTemplates(ctx, docker)
	if err != nil {
		return err
	}

	log.Info("building docker image")
	if err := imagers[docker.Use].Build(ctx, tmp, images, buildFlags); err != nil {
		if isFileNotFoundError(err.Error()) {
			var files []string
			_ = filepath.Walk(tmp, func(path string, info fs.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}
				files = append(files, info.Name())
				return nil
			})
			return fmt.Errorf(`seems like you tried to copy a file that is not available in the build context.

Here's more information about the build context:

dir: %q
files in that dir:
 %s

Previous error:
%w`, tmp, strings.Join(files, "\n "), err)
		}
		return err
	}

	for _, img := range images {
		ctx.Artifacts.Add(&artifact.Artifact{
			Type:   artifact.PublishableDockerImage,
			Name:   img,
			Path:   img,
			Goarch: docker.Goarch,
			Goos:   docker.Goos,
			Goarm:  docker.Goarm,
			Extra: map[string]interface{}{
				dockerConfigExtra: docker,
			},
		})
	}
	return nil
}

func isFileNotFoundError(out string) bool {
	if strings.Contains(out, `executable file not found in $PATH`) {
		return false
	}
	return strings.Contains(out, "file not found") ||
		strings.Contains(out, "not found: not found")
}

func processImageTemplates(ctx *context.Context, docker config.Docker) ([]string, error) {
	// nolint:prealloc
	var images []string
	for _, imageTemplate := range docker.ImageTemplates {
		image, err := tmpl.New(ctx).Apply(imageTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to execute image template '%s': %w", imageTemplate, err)
		}
		if image == "" {
			continue
		}

		images = append(images, image)
	}

	return images, nil
}

func processBuildFlagTemplates(ctx *context.Context, docker config.Docker) ([]string, error) {
	// nolint:prealloc
	var buildFlags []string
	for _, buildFlagTemplate := range docker.BuildFlagTemplates {
		buildFlag, err := tmpl.New(ctx).Apply(buildFlagTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to process build flag template '%s': %w", buildFlagTemplate, err)
		}
		buildFlags = append(buildFlags, buildFlag)
	}
	return buildFlags, nil
}

func dockerPush(ctx *context.Context, image *artifact.Artifact) error {
	log.WithField("image", image.Name).Info("pushing")

	docker, err := artifact.Extra[config.Docker](*image, dockerConfigExtra)
	if err != nil {
		return err
	}

	if strings.TrimSpace(docker.SkipPush) == "true" {
		return pipe.Skip("docker.skip_push is set: " + image.Name)
	}
	if strings.TrimSpace(docker.SkipPush) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' push, skipping docker publish: " + image.Name)
	}

	digest, err := imagers[docker.Use].Push(ctx, image.Name, docker.PushFlags)
	if err != nil {
		return err
	}

	art := &artifact.Artifact{
		Type:   artifact.DockerImage,
		Name:   image.Name,
		Path:   image.Path,
		Goarch: image.Goarch,
		Goos:   image.Goos,
		Goarm:  image.Goarm,
		Extra:  map[string]interface{}{},
	}
	if docker.ID != "" {
		art.Extra[artifact.ExtraID] = docker.ID
	}
	art.Extra[artifact.ExtraDigest] = digest

	ctx.Artifacts.Add(art)
	return nil
}
