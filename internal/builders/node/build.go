// Package node builds Node.js Single Executable Application (SEA)
// binaries.
//
// The pipeline shells out to a build-tool Node.js (≥ v25.5, downloaded
// once per host into the user cache) and invokes `node --build-sea
// sea-config.json` against the per-target Node binary GoReleaser
// fetches from https://nodejs.org/dist (verifying SHA-256). On macOS
// targets the produced Mach-O is ad-hoc signed via quill (pure-Go) so
// it loads on Apple Silicon out of the box; users with a Developer ID
// can layer real signing on top via the signs: pipe.
//
// Concurrent builds are enabled — each target runs --build-sea against
// its own scratch directory and outputs to its own path; nothing is
// shared across targets.
//
// Co-authored-by: Vedant Mohan Goyal <83997633+vedantmgoyal9@users.noreply.github.com>
//
// Builder skeleton and target list are derived from PR
// https://github.com/goreleaser/goreleaser/pull/6136 by @vedantmgoyal9.
package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/goreleaser/goreleaser/v2/internal/nodesea"
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
	_ api.PreparedBuilder   = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("node", Default)
}

// Builder is the Node.js SEA builder.
type Builder struct{}

// AllowConcurrentBuilds implements build.ConcurrentBuilder. Each
// per-target build runs `node --build-sea` against its own scratch
// directory and writes to its own output path; nothing is shared, so
// the builder is safe to run concurrently.
func (b *Builder) AllowConcurrentBuilds() bool { return true }

// Dependencies implements build.DependingBuilder. The new --build-sea
// flow auto-downloads its build-tool Node when needed, so no system
// `node` is strictly required. Returning "node" preserves the
// preflight hint goreleaser surfaces for users who'd rather provide
// their own.
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
	if build.Main == "" {
		build.Main = "index.js"
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

// Prepare implements build.PreparedBuilder. It runs once per build
// configuration before any per-target Build call: resolves and probes
// the build-tool Node up front (downloading it if necessary),
// validates that the resolved target Node version is in the
// V2-blob-format supported range, and runs `npm run build` when the
// project's `package.json` declares a `scripts.build` entry. Failing
// here is preferable to failing partway through a multi-target build.
func (b *Builder) Prepare(ctx *context.Context, build config.Build) error {
	nodePath, err := nodesea.BuildToolNode(ctx)
	if err != nil {
		return fmt.Errorf("nodesea: locate build-tool node: %w", err)
	}
	log.WithField("path", nodePath).Debug("resolved build-tool node")

	version, err := nodesea.ResolveVersion(build.Dir)
	if err != nil {
		return fmt.Errorf("nodesea: resolve target node version: %w", err)
	}
	log.WithField("version", version).Debug("resolved target node version")

	if err := nodesea.ValidateTargetNodeVersion(version); err != nil {
		return err
	}

	return runNPMBuildScript(ctx, build)
}

// runNPMBuildScript runs `npm run build` in build.Dir when the
// project's `package.json` declares a non-empty `scripts.build` entry,
// so the file referenced by `build.Main` is the freshly bundled
// output. Skipped silently when no such script is declared.
//
// Dependency installation (`npm ci` and friends) is intentionally not
// performed here — drive it from the `before:` hook instead.
func runNPMBuildScript(ctx *context.Context, build config.Build) error {
	env := append(os.Environ(), ctx.Env.Strings()...)
	tenv, err := base.TemplateEnv(build.Env, tmpl.New(ctx))
	if err != nil {
		return fmt.Errorf("nodesea: template env: %w", err)
	}
	env = append(env, tenv...)
	return nodesea.RunNPMBuild(ctx, build.Dir, env, logext.NewWriter(), logext.NewWriter())
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	target := nodedist.Target(t.Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   target.Goos(),
		Goarch: target.Goarch(),
		Target: t.Target,
		Extra: map[string]any{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "node",
		},
	}

	env := append([]string{}, ctx.Env.Strings()...)
	tpl := tmpl.New(ctx).WithBuildOptions(options).WithEnvS(env).WithArtifact(a)
	tenv, err := base.TemplateEnv(build.Env, tpl)
	if err != nil {
		return err
	}
	env = append(env, tenv...)

	log.WithField("binary", options.Name).
		WithField("target", options.Target.String()).
		Info("building")

	if err := buildViaBuildSEA(ctx, build, target, options, tpl); err != nil {
		return err
	}

	if err := base.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}

// buildViaBuildSEA dispatches to the `node --build-sea` flow
// (nodesea.BuildViaBuildSEA). The user's sea-config.json (if any) is
// read by the nodesea package directly from build.Dir.
func buildViaBuildSEA(
	ctx *context.Context,
	build config.Build,
	target nodedist.Target,
	options api.Options,
	tpl *tmpl.Template,
) error {
	main, err := tpl.Apply(build.Main)
	if err != nil {
		return fmt.Errorf("nodesea: template main: %w", err)
	}
	mainPath := filepath.Join(build.Dir, main)
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("nodesea: main %q not found in %q: %w", main, build.Dir, err)
	}

	version, err := nodesea.ResolveVersion(build.Dir)
	if err != nil {
		return fmt.Errorf("nodesea: resolve node version: %w", err)
	}

	buildToolNode, err := nodesea.BuildToolNode(ctx)
	if err != nil {
		return fmt.Errorf("nodesea: locate build-tool node: %w", err)
	}

	absMain, err := filepath.Abs(mainPath)
	if err != nil {
		absMain = mainPath
	}
	absBuildDir, err := filepath.Abs(build.Dir)
	if err != nil {
		absBuildDir = build.Dir
	}
	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return err
	}
	return nodesea.BuildViaBuildSEA(ctx, nodesea.BuildOptions{
		BuildToolNode: buildToolNode,
		Target:        target,
		Version:       version,
		MainJS:        absMain,
		OutPath:       options.Path,
		BuildDir:      absBuildDir,
	})
}

// currentTarget returns the nodejs.org/dist target identifier matching the
// machine running goreleaser, e.g. "linux-x64" or "darwin-arm64".
func currentTarget() string {
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
