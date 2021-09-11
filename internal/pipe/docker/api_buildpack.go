package docker

import (
	"context"
	"fmt"
	"strings"
)

type buildPackImager struct{}

func (i buildPackImager) Push(ctx context.Context, image string, flags []string) error {
	return dockerImager{}.Push(ctx, image, flags)
}

func (i buildPackImager) Build(ctx context.Context, root string, images, flags []string) error {
	if err := runCommand(ctx, "", "pack", i.buildCommand(images, flags)...); err != nil {
		return fmt.Errorf("failed to build %s: %w", images[0], err)
	}
	return nil
}

func (i buildPackImager) buildCommand(images, flags []string) []string {
	base := []string{"build", images[0]}
	for j := 1; j < len(images); j++ {
		base = append(base, "-t", images[j])
	}

	builderConfigured := false
	for _, flag := range flags {
		if strings.HasPrefix(flag, "-B") || strings.HasPrefix(flag, "--builder") {
			builderConfigured = true
		}
	}

	if !builderConfigured {
		flags = append(flags, "--builder=gcr.io/buildpacks/builder:v1")
	}

	return append(base, flags...)
}
