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

	env := ctx.Env.Strings()
	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithArtifact(a).
		WithEnvS(env)
	tenv, err := base.TemplateEnv(build.Env, tpl)
	if err != nil {
		return err
	}
	env = append(env, tenv...)

	tool, err := tpl.Apply(build.Tool)
	if err != nil {
		return fmt.Errorf("node: template tool: %w", err)
	}
	if err := checkHostNodeVersion(ctx, tool, env); err != nil {
		return err
	}

	targetNode, err := ensureNode(ctx, build.Dir, target.Target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return err
	}

	cfgPath := filepath.Join(filepath.Dir(options.Path), "sea-config.json")
	if err := createSEAConfig(tpl, build, cfgPath, targetNode, options.Path); err != nil {
		return err
	}

	log.WithField("binary", options.Name).
		WithField("target", options.Target.String()).
		Info("building")

	command := []string{tool, "--build-sea", cfgPath}
	if err := base.Exec(ctx, command, env, ""); err != nil {
		return err
	}

	if target.Goos() == "darwin" {
		if err := signMachO(options.Path, filepath.Base(options.Path)); err != nil {
			return err
		}
	}

	if err := base.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}
