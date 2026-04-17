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

	// Resolve config file and validate.
	seaCfgPath := filepath.Join(build.Dir, "sea-config.json")
	cfg, err := readSeaConfig(seaCfgPath)
	if err != nil {
		return fmt.Errorf("nodesea: %w", err)
	}
	if err := rejectIncompatibleSnapshot(cfg, target); err != nil {
		return err
	}

	// Generate or fetch cached blob.
	res, err := ensureBlob(ctx, build, env, seaCfgPath, cfg)
	if err != nil {
		return err
	}

	// Prepare host (download, copy, unsign).
	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return err
	}
	if _, err := nodesea.PrepareHost(ctx, res.version, target, options.Path); err != nil {
		return err
	}

	// Inject blob and flip sentinel.
	if err := nodesea.Inject(target, options.Path, res.bytes); err != nil {
		return fmt.Errorf("nodesea: inject %s: %w", target, err)
	}

	if err := os.Chmod(options.Path, 0o755); err != nil {
		return err
	}
	if err := base.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}

func ensureBlob(ctx *context.Context, build config.Build, env []string, seaCfgPath string, cfg *seaConfig) (blobResult, error) {
	key := build.ID + "\x00" + build.Dir
	blobMu.Lock()
	defer blobMu.Unlock()
	if r, ok := blobCache[key]; ok {
		return r, nil
	}

	version, source, err := nodesea.ResolveVersion(ctx, build.Dir, "")
	if err != nil {
		return blobResult{}, fmt.Errorf("nodesea: resolve node version: %w", err)
	}
	log.WithField("version", version).WithField("source", source).
		Info("resolved node version")

	if err := base.Exec(ctx, []string{"node", "--experimental-sea-config", filepath.Base(seaCfgPath)}, env, build.Dir); err != nil {
		return blobResult{}, fmt.Errorf("nodesea: generate blob: %w", err)
	}

	blobPath := filepath.Join(build.Dir, cfg.Output)
	bytes, err := os.ReadFile(blobPath)
	if err != nil {
		return blobResult{}, fmt.Errorf("nodesea: read generated blob %s: %w", blobPath, err)
	}

	res := blobResult{bytes: bytes, version: version}
	blobCache[key] = res
	return res, nil
}

type seaConfig struct {
	Main         string `json:"main"`
	Output       string `json:"output"`
	UseCodeCache bool   `json:"useCodeCache"` //nolint:tagliatelle // node SEA spec
	UseSnapshot  bool   `json:"useSnapshot"`  //nolint:tagliatelle // node SEA spec
}

func readSeaConfig(path string) (*seaConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c seaConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Main == "" {
		return nil, fmt.Errorf(`%s: missing "main"`, path)
	}
	if c.Output == "" {
		return nil, fmt.Errorf(`%s: missing "output"`, path)
	}
	return &c, nil
}

func rejectIncompatibleSnapshot(cfg *seaConfig, target nodesea.Target) error {
	if !cfg.UseCodeCache && !cfg.UseSnapshot {
		return nil
	}
	if string(target) == CurrentTarget() {
		return nil
	}
	return errors.New("nodesea: useCodeCache/useSnapshot are host-specific; remove them when cross-compiling")
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
