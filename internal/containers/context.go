package containers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type ImageBuildContext struct {
	ID         string
	BuildPath  string
	BuildFlags []string
	PushFlags  []string
	Platforms  []config.ContainerPlatform
	Images     []string
}

func (p ImageBuildContext) Log() *log.Entry {
	if len(p.Images) > 0 {
		return log.WithField("image", p.Images[0])
	} else {
		return log.WithField("id", p.ID)
	}
}

func LogEntry(ctx *context.Context, config config.Container) *log.Entry {
	images, _ := processImageTemplates(ctx, config.ImageTemplates)
	if len(images) > 0 {
		return log.WithField("image", images[0])
	} else {
		return log.WithField("id", config.ID)
	}
}

func BuildContext(ctx *context.Context, id string, imageDef config.ImageDefinition, platforms []config.ContainerPlatform) (ImageBuildContext, func(), error) {
	context := ImageBuildContext{
		ID:        id,
		Platforms: platforms,
	}

	images, err := processImageTemplates(ctx, imageDef.ImageTemplates)
	if err != nil {
		return context, nil, err
	}

	if len(images) == 0 {
		return context, nil, pipe.Skip("no image templates found")
	}
	context.Images = images

	tmp, err := os.MkdirTemp("", "goreleaserdocker")
	if err != nil {
		return context, nil, fmt.Errorf("failed to create temporary dir: %w", err)
	}
	context.BuildPath = tmp

	keepContext := false
	defer func() {
		if !keepContext {
			os.RemoveAll(tmp)
		}
	}()

	log := log.WithField("image", images[0])
	log.Debug("tempdir: " + tmp)

	// This will set all binaries for all architectures within the context
	// To ensure multi-arch builds can access the correct binaries, they are copied to $tmp/$TARGETPLATFORM/$name
	artifacts := getApplicableArtifacts(ctx, imageDef, platforms)
	if len(artifacts.List()) == 0 {
		log.Warn("no binaries or packages found for the given platform - COPY/ADD may not work")
	}
	log.WithField("artifacts", artifacts.Paths()).Debug("found artifacts")

	if err := tmpl.New(ctx).ApplyAll(
		&imageDef.Dockerfile,
	); err != nil {
		return context, nil, err
	}
	if err := gio.Copy(
		imageDef.Dockerfile,
		filepath.Join(tmp, "Dockerfile"),
	); err != nil {
		return context, nil, fmt.Errorf("failed to copy dockerfile: %w", err)
	}

	for _, file := range imageDef.Files {
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(file)), 0o755); err != nil {
			return context, nil, fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
		if err := gio.Copy(file, filepath.Join(tmp, file)); err != nil {
			return context, nil, fmt.Errorf("failed to copy extra file '%s': %w", file, err)
		}
	}

	for _, art := range artifacts.List() {
		target := filepath.Join(tmp, art.Goos, art.Goarch, art.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return context, nil, fmt.Errorf("failed to make dir for artifact: %w", err)
		}

		if err := gio.Copy(art.Path, target); err != nil {
			return context, nil, fmt.Errorf("failed to copy artifact: %w", err)
		}

		if len(context.Platforms) == 1 {
			if err := gio.Link(target, filepath.Join(tmp, art.Name)); err != nil {
				return context, nil, fmt.Errorf("failed to link artifact: %w", err)
			}
		}
	}

	buildFlags, err := processBuildFlagTemplates(ctx, imageDef.BuildFlagTemplates)
	if err != nil {
		return context, nil, err
	}
	context.BuildFlags = buildFlags
	context.PushFlags = imageDef.PushFlags

	keepContext = true
	return context, func() {
		os.RemoveAll(tmp)
	}, nil
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
