// Package node builds Node.js Single Executable Application (SEA)
// binaries.
//
// The pipeline shells out to whatever `node` is on `PATH` (must be
// ≥ v25.5 with LIEF) and invokes `node --build-sea sea-config.json`
// against the per-target Node binary GoReleaser fetches from
// https://nodejs.org/dist (verifying SHA-256). On macOS targets the
// produced Mach-O is ad-hoc signed via quill (pure-Go) so it loads on
// Apple Silicon out of the box; users with a Developer ID can layer
// real signing on top via the signs: pipe.
//
// Concurrent builds are enabled — each target runs --build-sea against
// its own scratch directory and outputs to its own path; nothing is
// shared across targets.
//
// Builder skeleton and target list are derived from PR
// https://github.com/goreleaser/goreleaser/pull/6136 by @vedantmgoyal9.
package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
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

// Dependencies implements build.DependingBuilder. The pipeline shells
// out to `node --build-sea`, so the host must have a `node` ≥ v25.5
// (LIEF-built) on PATH.
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

	if build.Tool == "" {
		build.Tool = "node"
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
		if _, err := b.Parse(t); err != nil {
			return build, err
		}
	}

	return build, nil
}

// Prepare implements build.PreparedBuilder. It runs once per build
// configuration before any per-target Build call: resolves the target
// Node version (failing fast if unset) and runs `npm run build` when
// the project's `package.json` declares a `scripts.build` entry. The
// host `node` is invoked as-is at Build time; if it cannot drive
// `--build-sea` the underlying error is surfaced to the user.
//
// Dependency installation (`npm ci` and friends) is intentionally not
// performed here — drive it from the `before:` hook instead.
func (b *Builder) Prepare(ctx *context.Context, build config.Build) error {
	version, err := resolveVersion(build.Dir)
	if err != nil {
		return fmt.Errorf("node: resolve target node version: %w", err)
	}
	log.WithField("version", version).Debug("resolved target node version")

	pkg, err := packagejson.Open(filepath.Join(build.Dir, "package.json"))
	if err != nil {
		return fmt.Errorf("node: scan package.json: %w", err)
	}
	if strings.TrimSpace(pkg.Scripts["build"]) == "" {
		log.WithField("dir", build.Dir).
			Debug("no scripts.build in package.json; skipping auto-bundle")
		return nil
	}

	env := append(os.Environ(), ctx.Env.Strings()...)
	tenv, err := base.TemplateEnv(build.Env, tmpl.New(ctx))
	if err != nil {
		return fmt.Errorf("node: template env: %w", err)
	}
	env = append(env, tenv...)
	log.WithField("dir", build.Dir).Info("running npm run build")
	return base.Exec(ctx, []string{"npm", "run", "build"}, env, build.Dir)
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	target := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   target.Goos(),
		Goarch: target.Goarch(),
		Target: target.Target,
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

// buildViaBuildSEA dispatches to buildSEA. The user's sea-config.json
// (if any) is read directly from build.Dir.
func buildViaBuildSEA(
	ctx *context.Context,
	build config.Build,
	target Target,
	options api.Options,
	tpl *tmpl.Template,
) error {
	main, err := tpl.Apply(build.Main)
	if err != nil {
		return fmt.Errorf("node: template main: %w", err)
	}
	mainPath := filepath.Join(build.Dir, main)
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("node: main %q not found in %q: %w", main, build.Dir, err)
	}

	absMain, err := filepath.Abs(mainPath)
	if err != nil {
		return fmt.Errorf("node: abs main %q: %w", mainPath, err)
	}
	absBuildDir, err := filepath.Abs(build.Dir)
	if err != nil {
		return fmt.Errorf("node: abs build dir %q: %w", build.Dir, err)
	}
	return buildSEA(ctx, target, build.Tool, absBuildDir, absMain, options.Path)
}
