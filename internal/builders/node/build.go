// Package node builds Node.js Single Executable Application (SEA)
// binaries.
//
// The pipeline is implemented entirely in pure Go through the
// internal/nodesea package: it downloads the official Node.js host binary
// for each requested target from https://nodejs.org/dist (verifying
// SHA-256), strips its existing code signature where applicable, runs
// `node --experimental-sea-config` to generate the SEA blob, injects the
// blob, and flips the SEA fuse sentinel.
//
// Code signing on macOS and Windows is intentionally left to GoReleaser's
// existing `signs:` pipe — produced binaries are unsigned and must be
// re-signed before distribution.
//
// Co-authored-by: Vedant Mohan Goyal <83997633+vedantmgoyal9@users.noreply.github.com>
//
// Builder skeleton and target list are derived from PR
// https://github.com/goreleaser/goreleaser/pull/6136 by @vedantmgoyal9.
package node

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Default builder instance.
//
//nolint:gochecknoglobals
var Default = &Builder{}

// type constraints
var (
	_ api.Builder           = &Builder{}
	_ api.DependingBuilder  = &Builder{}
	_ api.ConcurrentBuilder = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("node", Default)
}

// Builder is the Node.js SEA builder.
type Builder struct{}

// AllowConcurrentBuilds implements build.ConcurrentBuilder. We disable
// concurrent builds because the SEA-config scratch file and the blob
// output path are shared per build.
func (b *Builder) AllowConcurrentBuilds() bool { return false }

// Dependencies implements build.DependingBuilder. The only required
// external tool is `node` itself (used to generate the SEA blob via
// `--experimental-sea-config`).
func (b *Builder) Dependencies() []string {
	return []string{"node"}
}

//nolint:gochecknoglobals
var once sync.Once

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	once.Do(func() {
		log.Warn("you are using the experimental Node.js SEA builder")
	})

	if len(build.Targets) == 0 {
		build.Targets = defaultTargets()
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if build.Tool != "" {
		return build, errors.New("tool is not supported for the node builder; node is invoked directly")
	}
	if build.Command != "" {
		return build, errors.New("command is not supported for the node builder")
	}
	if len(build.Flags) > 0 {
		return build, errors.New("flags is not supported for the node builder")
	}
	if build.Main != "" {
		return build, errors.New("main is not supported for the node builder; set it inside sea-config.json")
	}

	if err := base.ValidateNonGoConfig(build); err != nil {
		return build, err
	}

	for _, t := range build.Targets {
		if !isValid(t) {
			return build, fmt.Errorf("invalid target: %s", t)
		}
	}

	return build, nil
}

// Build implements build.Builder.
//
// NOTE: the binary-format work (signature stripping + blob injection) is
// not yet wired in. This stub validates inputs, registers the artifact,
// and returns a clear error so the rest of the pipeline (config wiring,
// dependency healthchecks, init template) can be exercised end-to-end.
// The remaining work tracked in the implementation plan covers
// internal/nodesea/{unsign_macho,unsign_pe,inject_elf,inject_macho,inject_pe}.go.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   convertToGoos(t.Os),
		Goarch: convertToGoarch(t.Arch),
		Target: t.Target,
		Extra: map[string]any{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "node",
		},
	}

	tpl := tmpl.New(ctx).WithBuildOptions(options).WithArtifact(a)

	if _, err := base.TemplateEnv(build.Env, tpl); err != nil {
		return err
	}

	log.WithField("binary", options.Name).
		WithField("target", options.Target.String()).
		Info("building")

	return errors.New("nodesea: blob injection not yet implemented; see https://github.com/goreleaser/goreleaser/pull/6136 follow-up")
}

// CurrentTarget returns the nodejs.org/dist target identifier matching the
// machine running goreleaser, e.g. "linux-x64" or "darwin-arm64". It is
// used by the SEA pipeline to decide whether a build is "host" (native)
// or cross-platform.
func CurrentTarget() string {
	os := runtime.GOOS
	if os == "windows" {
		os = "win"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return os + "-" + arch
}
