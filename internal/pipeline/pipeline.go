// Package pipeline provides generic errors for pipes to use.
package pipeline

import (
	"fmt"

	"github.com/goreleaser/goreleaser/internal/pipe/announce"
	"github.com/goreleaser/goreleaser/internal/pipe/archive"
	"github.com/goreleaser/goreleaser/internal/pipe/aur"
	"github.com/goreleaser/goreleaser/internal/pipe/before"
	"github.com/goreleaser/goreleaser/internal/pipe/brew"
	"github.com/goreleaser/goreleaser/internal/pipe/build"
	"github.com/goreleaser/goreleaser/internal/pipe/changelog"
	"github.com/goreleaser/goreleaser/internal/pipe/checksums"
	"github.com/goreleaser/goreleaser/internal/pipe/chocolatey"
	"github.com/goreleaser/goreleaser/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/internal/pipe/dist"
	"github.com/goreleaser/goreleaser/internal/pipe/docker"
	"github.com/goreleaser/goreleaser/internal/pipe/effectiveconfig"
	"github.com/goreleaser/goreleaser/internal/pipe/env"
	"github.com/goreleaser/goreleaser/internal/pipe/git"
	"github.com/goreleaser/goreleaser/internal/pipe/gomod"
	"github.com/goreleaser/goreleaser/internal/pipe/krew"
	"github.com/goreleaser/goreleaser/internal/pipe/metadata"
	"github.com/goreleaser/goreleaser/internal/pipe/nfpm"
	"github.com/goreleaser/goreleaser/internal/pipe/nix"
	"github.com/goreleaser/goreleaser/internal/pipe/notary"
	"github.com/goreleaser/goreleaser/internal/pipe/partial"
	"github.com/goreleaser/goreleaser/internal/pipe/prebuild"
	"github.com/goreleaser/goreleaser/internal/pipe/publish"
	"github.com/goreleaser/goreleaser/internal/pipe/reportsizes"
	"github.com/goreleaser/goreleaser/internal/pipe/sbom"
	"github.com/goreleaser/goreleaser/internal/pipe/scoop"
	"github.com/goreleaser/goreleaser/internal/pipe/semver"
	"github.com/goreleaser/goreleaser/internal/pipe/sign"
	"github.com/goreleaser/goreleaser/internal/pipe/snapcraft"
	"github.com/goreleaser/goreleaser/internal/pipe/snapshot"
	"github.com/goreleaser/goreleaser/internal/pipe/sourcearchive"
	"github.com/goreleaser/goreleaser/internal/pipe/universalbinary"
	"github.com/goreleaser/goreleaser/internal/pipe/upx"
	"github.com/goreleaser/goreleaser/internal/pipe/winget"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Piper defines a pipe, which can be part of a pipeline (a series of pipes).
type Piper interface {
	fmt.Stringer

	// Run the pipe
	Run(ctx *context.Context) error
}

// BuildPipeline contains all build-related pipe implementations in order.
//
//nolint:gochecknoglobals
var BuildPipeline = []Piper{
	// load and validate environment variables
	env.Pipe{},
	// get and validate git repo state
	git.Pipe{},
	// parse current tag to a semver
	semver.Pipe{},
	// load default configs
	defaults.Pipe{},
	// setup things for partial builds/releases
	partial.Pipe{},
	// snapshot version handling
	snapshot.Pipe{},
	// run global hooks before build
	before.Pipe{},
	// ensure ./dist is clean
	dist.Pipe{},
	// setup metadata options
	metadata.Pipe{},
	// creates a metadta.json files in the dist directory
	metadata.MetaPipe{},
	// setup gomod-related stuff
	gomod.Pipe{},
	// run prebuild stuff
	prebuild.Pipe{},
	// proxy gomod if needed
	gomod.CheckGoModPipe{},
	// proxy gomod if needed
	gomod.ProxyPipe{},
	// writes the actual config (with defaults et al set) to dist
	effectiveconfig.Pipe{},
	// build
	build.Pipe{},
	// universal binary handling
	universalbinary.Pipe{},
	// notarize macos apps
	notary.MacOS{},
	// upx
	upx.Pipe{},
}

// BuildCmdPipeline is the pipeline run by goreleaser build.
//
//nolint:gochecknoglobals
var BuildCmdPipeline = append(
	BuildPipeline,
	reportsizes.Pipe{},
	metadata.ArtifactsPipe{},
)

// Pipeline contains all pipe implementations in order.
//
//nolint:gochecknoglobals
var Pipeline = append(
	BuildPipeline,
	// builds the release changelog
	changelog.Pipe{},
	// archive in tar.gz, zip or binary (which does no archiving at all)
	archive.Pipe{},
	// archive the source code using git-archive
	sourcearchive.Pipe{},
	// archive via fpm (deb, rpm) using "native" go impl
	nfpm.Pipe{},
	// archive via snapcraft (snap)
	snapcraft.Pipe{},
	// create SBOMs of artifacts
	sbom.Pipe{},
	// checksums of the files
	checksums.Pipe{},
	// sign artifacts
	sign.Pipe{},
	// create arch linux aur pkgbuild
	aur.Pipe{},
	// create nixpkgs
	nix.NewBuild(),
	// winget installers
	winget.Pipe{},
	// create brew tap
	brew.Pipe{},
	// krew plugins
	krew.Pipe{},
	// create scoop buckets
	scoop.Pipe{},
	// create chocolatey pkg and publish
	chocolatey.Pipe{},
	// reports artifacts sizes to the log and to artifacts.json
	reportsizes.Pipe{},
	// create and push docker images
	docker.Pipe{},
	// publishes artifacts
	publish.New(),
	// creates a artifacts.json files in the dist directory
	metadata.ArtifactsPipe{},
	// announce releases
	announce.Pipe{},
)
