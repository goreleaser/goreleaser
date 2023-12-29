package containers

import (
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func DefaultPlatforms(platforms []config.ContainerPlatform) []config.ContainerPlatform {
	if len(platforms) == 0 {
		platforms = make([]config.ContainerPlatform, 1)
	}
	for i := range platforms {
		DefaultPlatform(&platforms[i])
	}
	return platforms
}

func DefaultPlatform(platform *config.ContainerPlatform) {
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
