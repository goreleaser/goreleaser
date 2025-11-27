// Package healthcheck checks for missing binaries that the user needs to
// install.
package healthcheck

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/chocolatey"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/docker"
	dockerv2 "github.com/goreleaser/goreleaser/v2/internal/pipe/docker/v2"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/nix"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sbom"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sign"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/snapcraft"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// HealthChecker should be implemented by pipes that want checks.
type HealthChecker interface {
	fmt.Stringer

	// Healthcheck does checks that are more complex than [DependencyChecker],
	// and returns an error if they fail.
	Healthcheck(ctx *context.Context) error
}

// DependencyChecker should be implemented by pipes that want checks.
type DependencyChecker interface {
	fmt.Stringer

	// Dependencies return the binaries of the dependencies needed.
	Dependencies(ctx *context.Context) []string
}

// HealthCheckers is the list of health checkers.
//
//nolint:gochecknoglobals
var HealthCheckers = []HealthChecker{
	dockerv2.Base{},
}

// DependencyCheckers is the list of dependency checkers.
//
//nolint:gochecknoglobals
var DependencyCheckers = []DependencyChecker{
	system{},
	builds{},
	snapcraft.Pipe{},
	sign.Pipe{},
	sign.BinaryPipe{},
	sign.DockerPipe{},
	sbom.Pipe{},
	docker.Pipe{},
	docker.ManifestPipe{},
	dockerv2.Base{},
	chocolatey.Pipe{},
	nix.New(),
}

type system struct{}

func (system) String() string                         { return "system" }
func (system) Dependencies(*context.Context) []string { return []string{"git"} }

type builds struct{}

func (builds) String() string                             { return "build" }
func (builds) Dependencies(ctx *context.Context) []string { return build.Dependencies(ctx) }
