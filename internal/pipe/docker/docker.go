package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/deprecate"
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

	useBuildx     = "buildx"
	useDocker     = "docker"
	useBuildPacks = "buildpacks"
)

// Pipe for docker.
type Pipe struct{}

func (Pipe) String() string                 { return "docker images" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Dockers) == 0 }

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
		if docker.Dockerfile == "" {
			docker.Dockerfile = "Dockerfile"
		}
		if docker.Buildx {
			deprecate.Notice(ctx, "docker.use_buildx")
			if docker.Use == "" {
				docker.Use = useBuildx
			}
		}
		if docker.Use == "" {
			docker.Use = useDocker
		}
		if err := validateImager(docker.Use); err != nil {
			return err
		}
		for _, f := range docker.Files {
			if f == "." || strings.HasPrefix(f, ctx.Config.Dist) {
				return fmt.Errorf("invalid docker.files: can't be . or inside dist folder: %s", f)
			}
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
	images := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableDockerImage)).List()
	for _, image := range images {
		if err := dockerPush(ctx, image); err != nil {
			return err
		}
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, docker := range ctx.Config.Dockers {
		docker := docker
		g.Go(func() error {
			log.WithField("docker", docker).Debug("looking for artifacts matching")
			filters := []artifact.Filter{
				artifact.ByGoos(docker.Goos),
				artifact.ByGoarch(docker.Goarch),
				artifact.ByGoarm(docker.Goarm),
				artifact.Or(
					artifact.ByType(artifact.Binary),
					artifact.ByType(artifact.LinuxPackage),
				),
			}
			if len(docker.IDs) > 0 {
				filters = append(filters, artifact.ByIDs(docker.IDs...))
			}
			artifacts := ctx.Artifacts.Filter(artifact.And(filters...))
			log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")
			return process(ctx, docker, artifacts.List())
		})
	}
	return g.Wait()
}

func process(ctx *context.Context, docker config.Docker, artifacts []*artifact.Artifact) error {
	tmp, err := os.MkdirTemp(ctx.Config.Dist, "goreleaserdocker")
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

	if docker.Use != useBuildPacks {
		dockerfile, err := tmpl.New(ctx).Apply(docker.Dockerfile)
		if err != nil {
			return err
		}
		if err := gio.Copy(dockerfile, filepath.Join(tmp, "Dockerfile")); err != nil {
			return fmt.Errorf("failed to copy dockerfile: %w", err)
		}
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
		return err
	}

	if strings.TrimSpace(docker.SkipPush) == "true" {
		return pipe.Skip("docker.skip_push is set")
	}
	if ctx.SkipPublish {
		return pipe.ErrSkipPublishEnabled
	}
	if strings.TrimSpace(docker.SkipPush) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' push, skipping docker publish")
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
	docker := image.Extra[dockerConfigExtra].(config.Docker)
	if err := imagers[docker.Use].Push(ctx, image.Name, docker.PushFlags); err != nil {
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
	ctx.Artifacts.Add(art)
	return nil
}
