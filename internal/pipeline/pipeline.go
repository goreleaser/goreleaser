// Package pipeline provides generic errors for pipes to use.
package pipeline

import (
	"fmt"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/announce"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/archive"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/aur"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/aursources"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/before"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/brew"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/build"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/cask"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/changelog"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/checksums"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/chocolatey"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/dist"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/docker"
	dockerv2 "github.com/goreleaser/goreleaser/v2/internal/pipe/docker/v2"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/effectiveconfig"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/env"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/gomod"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/ko"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/krew"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/makeself"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/metadata"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/nfpm"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/nix"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/notary"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/partial"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/prebuild"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/publish"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/reportsizes"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sbom"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/scoop"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/semver"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sign"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/snapcraft"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/snapshot"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/sourcearchive"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/universalbinary"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/upx"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/winget"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
	// set default dist folder and remove it if `--clean` is set
	dist.CleanPipe{},
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
	// ensure ./dist exists and is empty
	dist.Pipe{},
	// setup metadata options
	metadata.Pipe{},
	// creates a metadata.json files in the dist directory
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
	// upx
	upx.Pipe{},
	// sign binaries
	sign.BinaryPipe{},
	// notarize macos apps
	notary.MacOS{},
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
	// create makeself self-extracting archives
	makeself.Pipe{},
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
	// create arch linux aur pkgbuild (sources)
	aursources.Pipe{},
	// create nixpkgs
	nix.New(),
	// winget installers
	winget.Pipe{},
	// homebrew formula
	brew.Pipe{},
	// homebrew cask
	cask.Pipe{},
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
	dockerv2.Snapshot{},
	// create and push docker images using ko
	ko.Pipe{},
	// publishes artifacts
	publish.New(),
	// creates a artifacts.json files in the dist directory
	metadata.ArtifactsPipe{},
	// announce releases
	announce.Pipe{},
)
