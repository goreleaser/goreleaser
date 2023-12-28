package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	containerbuilder "github.com/goreleaser/goreleaser/internal/pipe/container/builder"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe for containers.
type Pipe struct{}

func (Pipe) String() string { return "container images" }

func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Containers) == 0 || skips.Any(ctx, skips.Container)
}

func (Pipe) Dependencies(ctx *context.Context) []string {
	return []string{"docker"}
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("buildkits")
	for i := range ctx.Config.Containers {
		container := &ctx.Config.Containers[i]

		if container.ID != "" {
			ids.Inc(container.ID)
		}
		if len(container.Platforms) == 0 {
			container.Platforms = []config.ContainerPlatform{{
				Goos:   "linux",
				Goarch: "amd64",
			}}
		} else {
			for i := range container.Platforms {
				platform := &container.Platforms[i]
				if platform.Goos == "" {
					platform.Goos = "linux"
				}
				if platform.Goarch == "" {
					platform.Goarch = "amd64"
				}
				if platform.Goarm == "" {
					platform.Goarm = "6"
				}
				if platform.Goamd64 == "" {
					platform.Goamd64 = "v1"
				}
			}
		}
		if container.Dockerfile == "" {
			container.Dockerfile = "Dockerfile"
		}
	}
	return ids.Validate()
}

type runPhase = string

const (
	buildRun       = runPhase("buildRun")
	releaseRun     = runPhase("releaseRun")
	releasePublish = runPhase("releasePublish")
)

// Build and publish the docker images.
func (p Pipe) Publish(ctx *context.Context) error {
	return p.runContainerBuilds(ctx, releasePublish)
}

// Build the images only.
func (p Pipe) Run(ctx *context.Context) error {
	// if !skips.Any(ctx, skips.Publish) {
	// 	return pipe.Skip("images will directly be published later")
	// }

	// How to know if we are in build instead?
	return p.runContainerBuilds(ctx, releaseRun)
}

func (p Pipe) runContainerBuilds(ctx *context.Context, phase runPhase) error {
	containerSkips := pipe.SkipMemento{}
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for i, container := range ctx.Config.Containers {
		i := i
		containerConfig := container

		builder, err := containerbuilder.New(ctx, containerConfig.Builder)
		if err != nil {
			return fmt.Errorf("failed to get container builder for %s: %w", containerConfig.ID, err)
		}
		importImages, pushImages, skip, err := computeAction(ctx, phase, containerConfig.SkipImport, containerConfig.SkipPush, builder.SkipBuildIfPublish())
		if err != nil {
			return err
		}
		if skip != "" {
			images, _ := processImageTemplates(ctx, containerConfig.ImageTemplates)
			if len(images) > 0 {
				log.WithField("image", images[0]).Infof("skipping: %s", skip)
			} else {
				log.WithField("index", i).Infof("skipping: %s", skip)
			}
			containerSkips.Remember(pipe.Skip(skip))
			continue
		}

		g.Go(func() error {
			return process(ctx, containerConfig, importImages, pushImages, builder)
		})
	}
	if err := g.Wait(); err != nil {
		if pipe.IsSkip(err) {
			containerSkips.Remember(err)
		} else {
			return fmt.Errorf("docker build failed: %w", err)
		}
	}
	return containerSkips.Evaluate()
}

func computeAction(ctx *context.Context, phase runPhase, skipImport, skipPush string, skipBuildIfPublish bool) (importImages, pushImages bool, skip string, err error) {
	skipImport, err = tmpl.New(ctx).Apply(skipImport)
	if err != nil {
		return false, false, "", fmt.Errorf("failed to evaluate skipImport")
	}
	skipImport = strings.TrimSpace(skipImport)
	skipPush, err = tmpl.New(ctx).Apply(skipPush)
	if err != nil {
		return false, false, "", fmt.Errorf("failed to evaluate skipPush")
	}
	skipPush = strings.TrimSpace(skipPush)

	switch phase {
	case buildRun:
		// Build the image in all cases.
		// Import only if not snapshot or explicitly excluded
		if !ctx.Snapshot && skipImport != "true" {
			importImages = true
		}
	case releaseRun:
		// Build without pushing or importing images by default
		// Skip if the builder will build and push anyway
		if skipBuildIfPublish && !skips.Any(ctx, skips.Publish) {
			skip = "builder will directly publish the artifact"
		}
	case releasePublish:
		pushImages = true
		// If the image push is skipped, we only run it if we skipped the build
		if skipPush == "true" || ctx.Snapshot {
			if skipBuildIfPublish {
				pushImages = false
			} else {
				skip = "artifact push is skipped"
			}
		}
	}
	return importImages, pushImages, skip, nil
}

func getApplicableArtifacts(ctx *context.Context, imageDefinition config.ImageDefinition, platforms []config.ContainerPlatform) *artifact.Artifacts {
	var platformFilters []artifact.Filter
	for _, platform := range platforms {
		filters := []artifact.Filter{
			artifact.ByGoos(platform.Goos),
			artifact.ByGoarch(platform.Goarch),
		}
		switch platform.Goarch {
		case "amd64":
			filters = append(filters, artifact.ByGoamd64(platform.Goamd64))
		case "arm":
			filters = append(filters, artifact.ByGoarm(platform.Goarm))
		}
		platformFilters = append(platformFilters, artifact.And(
			filters...,
		))
	}
	filters := []artifact.Filter{
		artifact.Or(platformFilters...),
		artifact.Or(
			artifact.ByType(artifact.Binary),
			artifact.ByType(artifact.LinuxPackage),
		),
	}
	if len(imageDefinition.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(imageDefinition.IDs...))
	}
	return ctx.Artifacts.Filter(artifact.And(filters...))
}

func process(ctx *context.Context, config config.Container, importImages, pushImages bool, builder containerbuilder.ContainerBuilder) error {
	artifacts := getApplicableArtifacts(ctx, config.ImageDefinition, config.Platforms).List()
	if len(artifacts) == 0 {
		log.Warn("no binaries or packages found for the given platform - COPY/ADD may not work")
	}

	params := containerbuilder.ImageBuildParameters{
		ID:        config.ID,
		Platforms: config.Platforms,
	}

	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}
	defer os.RemoveAll(tmp)
	params.BuildPath = tmp

	images, err := processImageTemplates(ctx, config.ImageTemplates)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return pipe.Skip("no image templates found")
	}
	params.Images = images

	log := log.WithField("image", images[0])
	log.Debug("tempdir: " + tmp)

	if err := tmpl.New(ctx).ApplyAll(
		&config.Dockerfile,
	); err != nil {
		return err
	}
	if err := gio.Copy(
		config.Dockerfile,
		filepath.Join(tmp, "Dockerfile"),
	); err != nil {
		return fmt.Errorf("failed to copy dockerfile: %w", err)
	}

	for _, file := range config.Files {
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

	buildFlags, err := processBuildFlagTemplates(ctx, config.BuildFlagTemplates)
	if err != nil {
		return err
	}
	params.BuildFlags = buildFlags
	params.PushFlags = config.PushFlags

	return builder.Build(ctx, params, importImages, pushImages, log)
}

func processImageTemplates(ctx *context.Context, templates []string) ([]string, error) {
	// nolint:prealloc
	var images []string
	for _, imageTemplate := range templates {
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

func processBuildFlagTemplates(ctx *context.Context, flagTemplates []string) ([]string, error) {
	// nolint:prealloc
	var buildFlags []string
	for _, buildFlagTemplate := range flagTemplates {
		buildFlag, err := tmpl.New(ctx).Apply(buildFlagTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to process build flag template '%s': %w", buildFlagTemplate, err)
		}
		buildFlags = append(buildFlags, buildFlag)
	}
	return buildFlags, nil
}
