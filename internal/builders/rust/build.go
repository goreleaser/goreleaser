// Package rust builds rust binaries.
package rust

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
	"github.com/goreleaser/goreleaser/v2/internal/cargo"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
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
	_ api.PreparedBuilder   = &Builder{}
	_ api.ConcurrentBuilder = &Builder{}
	_ api.DependingBuilder  = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("rust", Default)
}

// Builder is golang builder.
type Builder struct{}

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"cargo", "rustup", "cargo-zigbuild", "zig"}
}

// AllowConcurrentBuilds implements build.ConcurrentBuilder.
func (b *Builder) AllowConcurrentBuilds() bool { return false }

// Prepare implements build.PreparedBuilder.
func (b *Builder) Prepare(ctx *context.Context, build config.Build) error {
	for _, target := range build.Targets {
		out, err := exec.CommandContext(ctx, "rustup", "target", "add", target).CombinedOutput()
		if err != nil {
			return fmt.Errorf("could not add target %s: %w: %s", target, err, string(out))
		}
	}
	return nil
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	parts := strings.Split(target, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	t := Target{
		Target: target,
		Os:     parts[2],
		Vendor: parts[1],
		Arch:   convertToGoarch(parts[0]),
	}

	if len(parts) > 3 {
		t.Abi = parts[3]
	}

	return t, nil
}

var once sync.Once

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	once.Do(func() {
		log.Warn("you are using the experimental Rust builder")
	})

	if len(build.Targets) == 0 {
		build.Targets = defaultTargets()
	}

	if build.Tool == "" {
		build.Tool = "cargo"
	}

	if build.Command == "" {
		build.Command = "zigbuild"
	}

	if len(build.Flags) == 0 {
		build.Flags = []string{"--release"}
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if build.Main != "" {
		return build, errors.New("main is not used for rust")
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
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	cargot, err := cargo.Open(filepath.Join(build.Dir, "Cargo.toml"))
	if err != nil {
		return err
	}
	// TODO: we should probably parse Cargo.toml and handle this better.
	// Go also has the possibility to build multiple binaries with a single
	// command, and we currently don't support that either.
	// We should build something generic enough for both cases, I think.
	if len(cargot.Workspace.Members) > 0 && !isSettingPackage(build.Flags) {
		return fmt.Errorf(
			"you need to specify which workspace to build, please add '--package=[name]' to your build flags, setting name to one of the available workspaces: %v",
			cargot.Workspace.Members[0],
		)
	}
	t := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   t.Os,
		Goarch: convertToGoarch(t.Arch),
		Target: t.Target,
		Extra: map[string]any{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "rust",
			keyAbi:                t.Abi,
		},
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	cargo, err := tpl.Apply(build.Tool)
	if err != nil {
		return err
	}

	command := []string{
		cargo,
		build.Command,
		"--target=" + t.Target,
	}

	tenv, err := base.TemplateEnv(build.Env, tpl)
	if err != nil {
		return err
	}
	env = append(env, tenv...)

	flags, err := tpl.Slice(build.Flags, tmpl.NonEmpty())
	if err != nil {
		return err
	}
	command = append(command, flags...)

	if err := base.Exec(ctx, command, env, build.Dir); err != nil {
		return err
	}

	realPath := filepath.Join(build.Dir, "target", t.Target, "release", options.Name)
	if err := gio.Copy(realPath, options.Path); err != nil {
		return err
	}

	if err := base.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}

func isSettingPackage(flags []string) bool {
	for _, flag := range flags {
		if strings.HasPrefix(flag, "-p=") ||
			strings.HasPrefix(flag, "--package=") {
			return true
		}
	}
	return false
}
