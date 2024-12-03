// Package healthcheck checks for missing binaries that the user needs to
// install.
package healthcheck

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/chocolatey"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/docker"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/nix"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sbom"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sign"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/snapcraft"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Healthchecker should be implemented by pipes that want checks.
type Healthchecker interface {
	fmt.Stringer

	// Dependencies return the binaries of the dependencies needed.
	Dependencies(ctx *context.Context) []string
}

// Healthcheckers is the list of healthchekers.
//
//nolint:gochecknoglobals
var Healthcheckers = []Healthchecker{
	system{},
	builds{},
	snapcraft.Pipe{},
	sign.Pipe{},
	sign.BinaryPipe{},
	sign.DockerPipe{},
	sbom.Pipe{},
	docker.Pipe{},
	docker.ManifestPipe{},
	chocolatey.Pipe{},
	nix.NewPublish(),
}

type system struct{}

func (system) String() string                         { return "system" }
func (system) Dependencies(*context.Context) []string { return []string{"git"} }

type builds struct{}

func (builds) String() string                             { return "build" }
func (builds) Dependencies(ctx *context.Context) []string { return build.Dependencies(ctx) }
