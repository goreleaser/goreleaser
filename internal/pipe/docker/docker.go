package docker

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/containers"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
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

func (Pipe) String() string { return "docker images" }

func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Dockers) == 0 || skips.Any(ctx, skips.Docker)
}

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
		containers.DefaultPlatform(&docker.ContainerPlatform)
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

			buildContext, cleanup, err := containers.BuildContext(ctx, docker.ID, docker.ImageDefinition, []config.ContainerPlatform{docker.ContainerPlatform})
			if err != nil {
				return err
			}
			defer cleanup()
			return process(ctx, buildContext, docker)
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

func process(ctx *context.Context, buildContext containers.ImageBuildContext, dockerConfig config.Docker) error {
	log := buildContext.Log()
	log.Info("building docker image")
	if err := imagers[dockerConfig.Use].Build(ctx, buildContext.BuildPath, buildContext.Images, buildContext.BuildFlags); err != nil {
		if isFileNotFoundError(err.Error()) {
			var files []string
			_ = filepath.Walk(buildContext.BuildPath, func(_ string, info fs.FileInfo, _ error) error {
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
%w`, buildContext.BuildPath, strings.Join(files, "\n "), err)
		}
		if isBuildxContextError(err.Error()) {
			return fmt.Errorf("docker buildx is not set to default context - please switch with 'docker context use default'")
		}
		return err
	}

	if len(buildContext.Platforms) != 1 {
		return fmt.Errorf("docker builder supports only single-platform builds")
	}
	platform := buildContext.Platforms[0]

	for _, img := range buildContext.Images {
		ctx.Artifacts.Add(&artifact.Artifact{
			Type:   artifact.PublishableDockerImage,
			Name:   img,
			Path:   img,
			Goarch: platform.Goarch,
			Goos:   platform.Goos,
			Goarm:  platform.Goarm,
			Extra: map[string]interface{}{
				dockerConfigExtra: dockerConfig,
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
		strings.Contains(out, ": not found")
}

func isBuildxContextError(out string) bool {
	return strings.Contains(out, "to switch to context")
}

func dockerPush(ctx *context.Context, image *artifact.Artifact) error {
	log.WithField("image", image.Name).Info("pushing")

	docker, err := artifact.Extra[config.Docker](*image, dockerConfigExtra)
	if err != nil {
		return err
	}

	skip, err := tmpl.New(ctx).Apply(docker.SkipPush)
	if err != nil {
		return err
	}
	if strings.TrimSpace(skip) == "true" {
		return pipe.Skip("docker.skip_push is set: " + image.Name)
	}
	if strings.TrimSpace(skip) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' push, skipping docker publish: " + image.Name)
	}

	digest, err := doPush(ctx, imagers[docker.Use], image.Name, docker.PushFlags)
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

func doPush(ctx *context.Context, img imager, name string, flags []string) (string, error) {
	var try int
	for try < 10 {
		digest, err := img.Push(ctx, name, flags)
		if err == nil {
			return digest, nil
		}
		if isRetryable(err) {
			log.WithField("try", try).
				WithField("image", name).
				WithError(err).
				Warnf("failed to push image, will retry")
			time.Sleep(time.Duration(try*10) * time.Second)
			try++
			continue
		}
		return "", fmt.Errorf("failed to push %s after %d tries: %w", name, try, err)
	}
	return "", nil // will never happen
}

func isRetryable(err error) bool {
	for _, code := range []int{
		http.StatusInternalServerError,
		// http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		// http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		// http.StatusInsufficientStorage,
		// http.StatusLoopDetected,
		http.StatusNotExtended,
		// http.StatusNetworkAuthenticationRequired,
	} {
		if strings.Contains(
			err.Error(),
			fmt.Sprintf(
				"received unexpected HTTP status: %d %s",
				code,
				http.StatusText(code),
			),
		) {
			return true
		}
	}
	return false
}
