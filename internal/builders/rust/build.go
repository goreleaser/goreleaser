package rust

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
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
		t.Environment = parts[3]
	}

	return t, nil
}

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	log.Warn("you are using the experimental Rust builder")

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

	if len(build.Ldflags) > 0 {
		return build, errors.New("ldflags is not used for rust")
	}

	if len(slices.Concat(
		build.Goos,
		build.Goarch,
		build.Goamd64,
		build.Go386,
		build.Goarm,
		build.Goarm64,
		build.Gomips,
		build.Goppc64,
		build.Goriscv64,
	)) > 0 {
		return build, errors.New("all go* fields are not used for rust, set targets instead")
	}

	if len(build.Ignore) > 0 {
		return build, errors.New("ignore is not used for rust, set targets instead")
	}

	if build.Buildmode != "" {
		return build, errors.New("buildmode is not used for rust")
	}

	if len(build.Tags) > 0 {
		return build, errors.New("tags is not used for rust")
	}

	if len(build.Asmflags) > 0 {
		return build, errors.New("asmflags is not used for rust")
	}

	if len(build.BuildDetailsOverrides) > 0 {
		return build, errors.New("overrides is not used for rust")
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
	if len(cargot.Workspace.Members) > 0 {
		return fmt.Errorf("goreleaser does not support cargo workspaces, please set the build 'dir' to one of the workspaces you want to build, e.g. 'dir: %q'", cargot.Workspace.Members[0])
	}
	t := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   t.Os,
		Goarch: convertToGoarch(t.Arch),
		Target: t.Target,
		Extra: map[string]interface{}{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "rust",
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

	for _, e := range build.Env {
		ee, err := tpl.Apply(e)
		if err != nil {
			return err
		}
		log.Debugf("env %q evaluated to %q", e, ee)
		if ee != "" {
			env = append(env, ee)
		}
	}

	tpl = tpl.WithEnvS(env)

	flags, err := tpl.Slice(build.Flags, tmpl.NonEmpty())
	if err != nil {
		return err
	}
	command = append(command, flags...)

	/* #nosec */
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env
	cmd.Dir = build.Dir
	log.Debug("running")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	if s := string(out); s != "" {
		log.WithField("cmd", command).Info(s)
	}

	if err := os.MkdirAll(filepath.Dir(options.Path), 0o755); err != nil {
		return err
	}
	realPath := filepath.Join(build.Dir, "target", t.Target, "release", options.Name)
	if err := gio.Copy(realPath, options.Path); err != nil {
		return err
	}

	// TODO: move this to outside builder for both go, rust, and zig
	modTimestamp, err := tpl.Apply(build.ModTimestamp)
	if err != nil {
		return err
	}
	if err := gio.Chtimes(a.Path, modTimestamp); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}
