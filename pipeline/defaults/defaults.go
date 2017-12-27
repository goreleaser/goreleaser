// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/artifactory"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/checksums"
	"github.com/goreleaser/goreleaser/pipeline/docker"
	"github.com/goreleaser/goreleaser/pipeline/fpm"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/sign"
	"github.com/goreleaser/goreleaser/pipeline/snapcraft"
	"github.com/goreleaser/goreleaser/pipeline/snapshot"
)

// Pipe that sets the defaults
type Pipe struct{}

func (Pipe) String() string {
	return "setting defaults for:"
}

var defaulters = []pipeline.Defaulter{
	snapshot.Pipe{},
	release.Pipe{},
	archive.Pipe{},
	build.Pipe{},
	fpm.Pipe{},
	snapcraft.Pipe{},
	checksums.Pipe{},
	sign.Pipe{},
	docker.Pipe{},
	artifactory.Pipe{},
	brew.Pipe{},
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Dist == "" {
		ctx.Config.Dist = "dist"
	}
	for _, defaulter := range defaulters {
		log.Info(defaulter.String())
		if err := defaulter.Default(ctx); err != nil {
			return err
		}
	}
	if ctx.Config.ProjectName == "" {
		ctx.Config.ProjectName = ctx.Config.Release.GitHub.Name
	}
	return nil
}
