// Package defaults implements the Pipe interface providing default values
// for missing configuration.
package defaults

import (
	"fmt"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/pipe/archive"
	"github.com/goreleaser/goreleaser/internal/pipe/artifactory"
	"github.com/goreleaser/goreleaser/internal/pipe/brew"
	"github.com/goreleaser/goreleaser/internal/pipe/build"
	"github.com/goreleaser/goreleaser/internal/pipe/checksums"
	"github.com/goreleaser/goreleaser/internal/pipe/docker"
	"github.com/goreleaser/goreleaser/internal/pipe/env"
	"github.com/goreleaser/goreleaser/internal/pipe/nfpm"
	"github.com/goreleaser/goreleaser/internal/pipe/project"
	"github.com/goreleaser/goreleaser/internal/pipe/release"
	"github.com/goreleaser/goreleaser/internal/pipe/s3"
	"github.com/goreleaser/goreleaser/internal/pipe/scoop"
	"github.com/goreleaser/goreleaser/internal/pipe/sign"
	"github.com/goreleaser/goreleaser/internal/pipe/snapcraft"
	"github.com/goreleaser/goreleaser/internal/pipe/snapshot"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that sets the defaults
type Pipe struct{}

func (Pipe) String() string {
	return "setting defaults"
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
		log.Debug(defaulter.String())
		if err := defaulter.Default(ctx); err != nil {
			return err
		}
	}
	return nil
}
