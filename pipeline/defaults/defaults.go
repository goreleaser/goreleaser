// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"fmt"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/artifactory"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/checksums"
	"github.com/goreleaser/goreleaser/pipeline/docker"
	"github.com/goreleaser/goreleaser/pipeline/env"
	"github.com/goreleaser/goreleaser/pipeline/fpm"
	"github.com/goreleaser/goreleaser/pipeline/nfpm"
	"github.com/goreleaser/goreleaser/pipeline/project"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/s3"
	"github.com/goreleaser/goreleaser/pipeline/scoop"
	"github.com/goreleaser/goreleaser/pipeline/sign"
	"github.com/goreleaser/goreleaser/pipeline/snapcraft"
	"github.com/goreleaser/goreleaser/pipeline/snapshot"
)

// Pipe that sets the defaults
type Pipe struct{}

func (Pipe) String() string {
	return "setting defaults for:"
}

// Defaulter can be implemented by a Piper to set default values for its
// configuration.
type Defaulter interface {
	fmt.Stringer

	// Default sets the configuration defaults
	Default(ctx *context.Context) error
}

var defaulters = []Defaulter{
	env.Pipe{},
	snapshot.Pipe{},
	release.Pipe{},
	project.Pipe{},
	archive.Pipe{},
	build.Pipe{},
	fpm.Pipe{},
	nfpm.Pipe{},
	snapcraft.Pipe{},
	checksums.Pipe{},
	sign.Pipe{},
	docker.Pipe{},
	artifactory.Pipe{},
	s3.Pipe{},
	brew.Pipe{},
	scoop.Pipe{},
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Dist == "" {
		ctx.Config.Dist = "dist"
	}
	if ctx.Config.GitHubURLs.Download == "" {
		ctx.Config.GitHubURLs.Download = "https://github.com"
	}
	for _, defaulter := range defaulters {
		log.Info(defaulter.String())
		if err := defaulter.Default(ctx); err != nil {
			return err
		}
	}
	return nil
}
