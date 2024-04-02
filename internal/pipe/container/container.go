package container

import (
	"fmt"
	"strings"

	"github.com/goreleaser/goreleaser/internal/containers"
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
		container.Platforms = containers.DefaultPlatforms(container.Platforms)
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
	var phase runPhase
	switch ctx.Action {
	case context.ActionBuild:
		phase = buildRun
	case context.ActionRelease:
		phase = releaseRun
	default:
		return pipe.Skip("nothing to build for this action")
	}
	return p.runContainerBuilds(ctx, phase)
}

func (p Pipe) runContainerBuilds(ctx *context.Context, phase runPhase) error {
	containerSkips := pipe.SkipMemento{}
	g := semerrgroup.NewSkipAware(semerrgroup.New(ctx.Parallelism))
	for _, container := range ctx.Config.Containers {
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
			log := containers.LogEntry(ctx, containerConfig)
			log.Infof("skipping: %s", skip)
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

func process(ctx *context.Context, config config.Container, importImages, pushImages bool, builder containerbuilder.ContainerBuilder) error {
	log := containers.LogEntry(ctx, config)

	context, cleanup, err := containers.BuildContext(ctx, config.ID, config.ImageDefinition, config.Platforms)
	if err != nil {
		if pipe.IsSkip(err) {
			log.Infof("skipping: %s", err)
		}
		return nil
	}
	defer cleanup()

	return builder.Build(ctx, context, importImages, pushImages, log)
}
