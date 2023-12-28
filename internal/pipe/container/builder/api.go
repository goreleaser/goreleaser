package builder

import (
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type ImageBuildParameters struct {
	ID         string
	BuildPath  string
	BuildFlags []string
	PushFlags  []string
	Platforms  []config.ContainerPlatform
	Images     []string
}

type ContainerBuilder interface {
	Build(ctx *context.Context, params ImageBuildParameters, importImages bool, pushImages bool, logger *log.Entry) error
	// If returning true, Build should only be called once (in Run for `goreleaser build` or `goreleaser release --snapshot`, in Publish for `goreleaser release`)
	// If returning false, Build should be called once in Run phase and again in Publish phase
	SkipBuildIfPublish() bool
}

func New(ctx *context.Context, builderConfig config.ContainerBuilder) (ContainerBuilder, error) {
	if builderConfig.BuildKit != nil {
		return buildKitBuilder(ctx, *builderConfig.BuildKit)
	}
	// Default to the current buildkit builder
	return buildKitBuilder(ctx, config.BuildKitBuilder{})
}
