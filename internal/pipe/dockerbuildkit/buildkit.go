package buildkit

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	buildkitConfigExtra = "BuildKitConfig"
)

// Pipe for docker buildkit.
type Pipe struct{}

func (Pipe) String() string { return "docker buildkit images" }

func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.DockerBuildKits) == 0 || skips.Any(ctx, skips.Docker)
}

func (Pipe) Dependencies(ctx *context.Context) []string {
	return []string{"docker"}
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("buildkits")
	for i := range ctx.Config.DockerBuildKits {
		docker := &ctx.Config.DockerBuildKits[i]

		if docker.ID != "" {
			ids.Inc(docker.ID)
		}
		if len(docker.Platforms) == 0 {
			docker.Platforms = []config.DockerPlatform{{
				Os:   "linux",
				Arch: "amd64",
			}}
		}
		if docker.Dockerfile == "" {
			docker.Dockerfile = "Dockerfile"
		}
	}
	return ids.Validate()
}

// Build and publish the docker images.
func (p Pipe) Publish(ctx *context.Context) error {
	return p.runBuildKitBuilds(ctx, false)
}

// Build the images only.
func (p Pipe) Run(ctx *context.Context) error {
	if !skips.Any(ctx, skips.Publish) {
		return pipe.Skip("buildkit will directly publish the artifact")
	}

	return p.runBuildKitBuilds(ctx, true)
}

func (p Pipe) runBuildKitBuilds(ctx *context.Context, buildOnly bool) error {
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for i, docker := range ctx.Config.DockerBuildKits {
		i := i
		dockerConfig := docker
		// If the command is build and we don't skip import, fallback to single platform builds
		// as buildx does not support `docker` output with multi-platform builds
		if buildOnly && !dockerConfig.SkipImport {
			for _, platform := range dockerConfig.Platforms {
				g.Go(func() error {
					log := log.WithField("index", i)
					log = log.WithField("platform", platform)
					return runSinglePlatformBuilds(ctx, dockerConfig, platform, log)
				})
			}
		} else {
			g.Go(func() error {
				return runMultiPlatformBuild(ctx, buildOnly, dockerConfig, log.WithField("index", i))
			})
		}
	}
	if err := g.Wait(); err != nil {
		if pipe.IsSkip(err) {
			return err
		}
		return fmt.Errorf("docker build failed: %w\nLearn more at https://goreleaser.com/errors/docker-build\n", err) // nolint:revive
	}
	return nil
}

// Currently buildkit does not support building multiple images and returning it to the daemon with --load
// In this case we run n parallel builds
func runSinglePlatformBuilds(ctx *context.Context, dockerConfig config.DockerBuildKit, platform config.DockerPlatform, log *log.Entry) error {
	log.Debug("looking for artifacts matching")
	filters := []artifact.Filter{
		artifact.ByGoos(platform.Os),
		artifact.ByGoarch(platform.Arch),
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.LinuxPackage),
		),
	}
	if len(dockerConfig.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(dockerConfig.IDs...))
	}
	artifacts := ctx.Artifacts.Filter(artifact.And(filters...))
	log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")
	return process(ctx, true, dockerConfig, []config.DockerPlatform{platform}, artifacts.List())
}

func runMultiPlatformBuild(ctx *context.Context, buildOnly bool, dockerConfig config.DockerBuildKit, log *log.Entry) error {
	var platformFilters []artifact.Filter
	for _, platform := range dockerConfig.Platforms {
		platformFilters = append(platformFilters, artifact.And(
			artifact.ByGoos(platform.Os),
			artifact.ByGoarch(platform.Arch),
		))
	}
	filters := []artifact.Filter{
		artifact.Or(platformFilters...),
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.LinuxPackage),
		),
	}
	if len(dockerConfig.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(dockerConfig.IDs...))
	}
	artifacts := ctx.Artifacts.Filter(artifact.And(filters...))
	log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")
	return process(ctx, buildOnly, dockerConfig, dockerConfig.Platforms, artifacts.List())
}

func process(ctx *context.Context, buildOnly bool, dockerConfig config.DockerBuildKit, platforms []config.DockerPlatform, artifacts []*artifact.Artifact) error {
	if len(artifacts) == 0 {
		log.Warn("no binaries or packages found for the given platform - COPY/ADD may not work")
	}
	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	images, err := processImageTemplates(ctx, dockerConfig)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return pipe.Skip("no image templates found")
	}

	log := log.WithField("image", images[0])
	log.Debug("tempdir: " + tmp)

	if err := tmpl.New(ctx).ApplyAll(
		&dockerConfig.Dockerfile,
	); err != nil {
		return err
	}
	if err := gio.Copy(
		dockerConfig.Dockerfile,
		filepath.Join(tmp, "Dockerfile"),
	); err != nil {
		return fmt.Errorf("failed to copy dockerfile: %w", err)
	}

	for _, file := range dockerConfig.Files {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0o755); err != nil {
			return fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
		if err := gio.Copy(file, filepath.Join(tmp, file)); err != nil {
			return fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
	}
	for _, art := range artifacts {
		target := filepath.Join(tmp, art.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("failed to make dir for artifact: %w", err)
		}

		if err := gio.Copy(art.Path, target); err != nil {
			return fmt.Errorf("failed to copy artifact: %w", err)
		}
	}

	buildFlags, err := processBuildFlagTemplates(ctx, dockerConfig)
	if err != nil {
		return err
	}

	var digest string
	if buildOnly {
		log.Info("building docker image")
		digest, err = build(ctx, tmp, images, buildFlags, dockerConfig.BuilderName, platforms, !dockerConfig.SkipImport)
	} else {
		log.Info("pushing docker image")
		var flags []string
		flags = append(flags, buildFlags...)
		flags = append(flags, dockerConfig.PushFlags...)
		digest, err = push(ctx, tmp, images, flags, dockerConfig.BuilderName, platforms)
	}

	if err != nil {
		if isFileNotFoundError(err.Error()) {
			var files []string
			_ = filepath.Walk(tmp, func(_ string, info fs.FileInfo, _ error) error {
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
		if isBuildxContextError(err.Error()) {
			return fmt.Errorf("docker buildx is not set to default context - please switch with 'docker context use default'")
		}
		return err
	}

	if digest != "" {
		for _, img := range images {
			art := &artifact.Artifact{
				Type:   artifact.DockerImage,
				Name:   img,
				Path:   img,
				Goarch: platforms[0].Arch,
				Goos:   platforms[0].Os,
				Extra: map[string]interface{}{
					buildkitConfigExtra:  dockerConfig,
					artifact.ExtraDigest: digest,
				},
			}
			if dockerConfig.ID != "" {
				art.Extra[artifact.ExtraID] = dockerConfig.ID
			}

			if buildOnly && !dockerConfig.SkipImport {
				art.Type = artifact.DockerImage
				art.Goarch = platforms[0].Arch
				art.Goos = platforms[0].Os
				ctx.Artifacts.Add(art)
			} else if !buildOnly {
				art.Type = artifact.DockerManifest
				ctx.Artifacts.Add(art)
			}
		}
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

func processImageTemplates(ctx *context.Context, docker config.DockerBuildKit) ([]string, error) {
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

func processBuildFlagTemplates(ctx *context.Context, docker config.DockerBuildKit) ([]string, error) {
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
