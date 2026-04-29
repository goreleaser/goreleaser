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
	"encoding/json"
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

// blobCache memoizes per-build (id, dir) blob generation so we don't
// invoke `node --experimental-sea-config` once per target.
//
//nolint:gochecknoglobals
var (
	blobMu    sync.Mutex
	blobCache = map[string]blobResult{}
)

type blobResult struct {
	bytes   []byte
	version string
}

// FeatureFlagLegacyInjector restores the pre-v25.5 in-process binary
// surgery code path that was the default before phase 3 of the
// migration. It exists as a temporary escape hatch and will be removed
// in a future release; new builds should use the default `--build-sea`
// flow.
const FeatureFlagLegacyInjector = "GORELEASER_NODE_LEGACY_INJECTOR"

// useLegacyInjector reports whether the user opted out of the
// `--build-sea` flow via the legacy-injector escape hatch.
func useLegacyInjector() bool {
	return os.Getenv(FeatureFlagLegacyInjector) == "1"
}

// Prepare implements build.PreparedBuilder. It runs once per build
// configuration before any per-target Build call. For the default
// `--build-sea` path we resolve and probe the build-tool Node up
// front (downloading it if necessary) and validate that the resolved
// target Node version is in the V2-blob-format supported range.
//
// The legacy injector path skips this preflight entirely.
func (b *Builder) Prepare(ctx *context.Context, build config.Build) error {
	if useLegacyInjector() {
		return nil
	}

	nodePath, err := nodesea.BuildToolNode(ctx)
	if err != nil {
		return fmt.Errorf("nodesea: locate build-tool node: %w", err)
	}
	log.WithField("path", nodePath).Debug("resolved build-tool node")

	version, source, err := nodesea.ResolveVersion(ctx, build.Dir, "")
	if err != nil {
		return fmt.Errorf("nodesea: resolve target node version: %w", err)
	}
	log.WithField("version", version).WithField("source", source).
		Debug("resolved target node version")

	if err := nodesea.ValidateTargetNodeVersion(version); err != nil {
		return err
	}
	return nil
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	target := nodesea.Target(t.Target)
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

	if useLegacyInjector() {
		// Legacy path: generate or fetch cached blob, then inject.
		res, err := ensureBlob(ctx, build, env)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
			return err
		}
		if err := nodesea.Build(ctx, res.version, target, options.Path, res.bytes); err != nil {
			return fmt.Errorf("nodesea: build %s: %w", target, err)
		}
	} else if err := buildViaBuildSEA(ctx, build, target, options); err != nil {
		return err
	}

	if err := base.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}

// buildViaBuildSEA dispatches to the new `node --build-sea` flow
// (nodesea.BuildViaBuildSEA), passing through the user-supplied
// SEAConfig from build configuration.
func buildViaBuildSEA(
	ctx *context.Context,
	build config.Build,
	target nodesea.Target,
	options api.Options,
) error {
	mainPath := filepath.Join(build.Dir, build.Main)
	if _, err := os.Stat(mainPath); err != nil {
		return fmt.Errorf("nodesea: main %q not found in %q: %w", build.Main, build.Dir, err)
	}

	version, source, err := nodesea.ResolveVersion(ctx, build.Dir, "")
	if err != nil {
		return fmt.Errorf("nodesea: resolve node version: %w", err)
	}
	log.WithField("version", version).WithField("source", source).
		Info("resolved node version")

	buildToolNode, err := nodesea.BuildToolNode(ctx)
	if err != nil {
		return fmt.Errorf("nodesea: locate build-tool node: %w", err)
	}

	absMain, err := filepath.Abs(mainPath)
	if err != nil {
		absMain = mainPath
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
		SEAConfig:     toNodeseaSEAConfig(build.SEAConfig),
	})
}

// toNodeseaSEAConfig translates the user-facing config.NodeSEAConfig
// into the internal nodesea.SEAConfig payload. Kept separate so the
// public yaml schema and the internal call site can evolve
// independently.
func toNodeseaSEAConfig(c config.NodeSEAConfig) nodesea.SEAConfig {
	return nodesea.SEAConfig{
		Assets:                        c.Assets,
		ExecArgv:                      c.ExecArgv,
		DisableExperimentalSEAWarning: c.DisableExperimentalSEAWarning,
		MainFormat:                    c.MainFormat,
	}
}

func ensureBlob(ctx *context.Context, build config.Build, env []string) (blobResult, error) {
	key := build.ID + "\x00" + build.Dir
	blobMu.Lock()
	defer blobMu.Unlock()
	if r, ok := blobCache[key]; ok {
		return r, nil
	}

	if _, err := os.Stat(filepath.Join(build.Dir, build.Main)); err != nil {
		return blobResult{}, fmt.Errorf("nodesea: main %q not found in %q: %w", build.Main, build.Dir, err)
	}

	version, source, err := nodesea.ResolveVersion(ctx, build.Dir, "")
	if err != nil {
		return blobResult{}, fmt.Errorf("nodesea: resolve node version: %w", err)
	}
	log.WithField("version", version).WithField("source", source).
		Info("resolved node version")

	scratch, err := os.MkdirTemp(ctx.Config.Dist, "node-sea-*")
	if err != nil {
		return blobResult{}, fmt.Errorf("nodesea: create scratch dir: %w", err)
	}
	defer os.RemoveAll(scratch)

	cfgPath := filepath.Join(scratch, "sea-config.json")
	blobPath := filepath.Join(scratch, "sea-prep.blob")
	cfg := map[string]any{
		"main":                          build.Main,
		"output":                        blobPath,
		"disableExperimentalSEAWarning": true,
	}
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return blobResult{}, err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o600); err != nil {
		return blobResult{}, fmt.Errorf("nodesea: write sea-config: %w", err)
	}

	absCfg, err := filepath.Abs(cfgPath)
	if err != nil {
		absCfg = cfgPath
	}
	if err := base.Exec(ctx, []string{"node", "--experimental-sea-config", absCfg}, env, build.Dir); err != nil {
		return blobResult{}, fmt.Errorf("nodesea: generate blob: %w", err)
	}

	bytes, err := os.ReadFile(blobPath)
	if err != nil {
		return blobResult{}, fmt.Errorf("nodesea: read generated blob %s: %w", blobPath, err)
	}

	res := blobResult{bytes: bytes, version: version}
	blobCache[key] = res
	return res, nil
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
