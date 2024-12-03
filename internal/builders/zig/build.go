package zig

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
	_ api.Builder          = &Builder{}
	_ api.DependingBuilder = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("zig", Default)
}

// Builder is golang builder.
type Builder struct{}

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"zig"}
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	parts := strings.Split(target, "-")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	t := Target{
		Target: target,
		Os:     convertToGoos(parts[1]),
		Arch:   convertToGoarch(parts[0]),
	}

	if len(parts) > 2 {
		t.Abi = parts[2]
	}

	return t, nil
}

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	log.Warn("you are using the experimental Zig builder")

	if len(build.Targets) == 0 {
		build.Targets = defaultTargets()
	}

	if build.GoBinary == "" {
		build.GoBinary = "zig"
	}

	if build.Command == "" {
		build.Command = "build"
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if build.Main != "" {
		return build, errors.New("main is not used for zig")
	}

	if len(build.Ldflags) > 0 {
		return build, errors.New("ldflags is not used for zig")
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
		return build, errors.New("all go* fields are not used for zig, set targets instead")
	}

	if len(build.Ignore) > 0 {
		return build, errors.New("ignore is not used for zig, set targets instead")
	}

	if build.Buildmode != "" {
		return build, errors.New("buildmode is not used for zig")
	}

	if len(build.Tags) > 0 {
		return build, errors.New("tags is not used for zig")
	}

	if len(build.Asmflags) > 0 {
		return build, errors.New("asmtags is not used for zig")
	}

	if len(build.BuildDetailsOverrides) > 0 {
		return build, errors.New("overrides is not used for zig")
	}

	for _, t := range build.Targets {
		switch checkTarget(t) {
		case targetValid:
			// lfg
		case targetBroken:
			log.Warnf("target might not be supported: %s", t)
		case targetInvalid:
			return build, fmt.Errorf("invalid target: %s", t)
		}
	}

	return build, nil
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	t := options.Target.(Target)
	a := &artifact.Artifact{
		Type:   artifact.Binary,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   convertToGoos(t.Os),
		Goarch: convertToGoarch(t.Arch),
		Target: t.Target,
		Extra: map[string]interface{}{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "zig",
		},
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	zigbin, err := tpl.Apply(build.GoBinary)
	if err != nil {
		return err
	}

	prefix := filepath.Join("zig-out", t.Target)
	command := []string{
		zigbin,
		build.Command,
		"-Dtarget=" + t.Target,
		"-p", prefix,
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

	flags, err := processFlags(tpl, build.Flags)
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
	realPath := filepath.Join(build.Dir, prefix, "bin", options.Name)
	if err := gio.Copy(realPath, options.Path); err != nil {
		return err
	}

	// TODO: move this to outside builder for both go and zig
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

func processFlags(tpl *tmpl.Template, flags []string) ([]string, error) {
	var processed []string
	for _, rawFlag := range flags {
		flag, err := tpl.Apply(rawFlag)
		if err != nil {
			return nil, err
		}
		if flag == "" {
			continue
		}
		processed = append(processed, flag)
	}
	return processed, nil
}
