package bun

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/common"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Default builder instance.
//
//nolint:gochecknoglobals
var Default = &Builder{}

var (
	_ api.Builder          = &Builder{}
	_ api.DependingBuilder = &Builder{}
)

//nolint:gochecknoinits
func init() {
	api.Register("bun", Default)
}

// Builder is bun builder.
type Builder struct{}

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"bun"}
}

// Target represents a build target.
type Target struct {
	Target string
	Os     string
	Arch   string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	target = strings.TrimPrefix(target, "bun-")
	parts := strings.Split(target, "-")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	// TODO: handle -modern and -baseline
	return Target{
		Target: "bun-" + target,
		Os:     parts[0],
		Arch:   parts[1],
	}, nil
}

var once sync.Once

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	once.Do(func() {
		log.Warn("you are using the experimental Bun builder")
	})

	if len(build.Targets) == 0 {
		build.Targets = []string{
			"linux-x64",
			"linux-arm64",
			"darwin-x64",
			"darwin-arm64",
			"windows-x86",
		}
	}

	if build.Tool == "" {
		build.Tool = "bun"
	}

	if build.Command == "" {
		build.Command = "build"
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if build.Main == "" {
		return build, errors.New("main is not used for bun")
	}

	if err := common.ValidateNonGoConfig(build); err != nil {
		return build, err
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
		Goos:   t.Os,
		Goarch: t.Arch,
		Target: t.Target,
		Extra: map[string]interface{}{
			artifact.ExtraBinary:  strings.TrimSuffix(filepath.Base(options.Path), options.Ext),
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "bun",
		},
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	bun, err := tpl.Apply(build.Tool)
	if err != nil {
		return err
	}

	command := []string{
		bun,
		build.Command,
		"--compile",
		"--target",
		t.Target,
		"--outfile",
		options.Path,
		build.Dir,
	}

	tenv, err := common.TemplateEnv(build, tpl)
	if err != nil {
		return err
	}
	env = append(env, tenv...)

	flags, err := tpl.Slice(build.Flags, tmpl.NonEmpty())
	if err != nil {
		return err
	}
	command = append(command, flags...)

	if err := common.Exec(ctx, command, env, build.Dir); err != nil {
		return err
	}

	if err := common.ChTimes(build, tpl, a); err != nil {
		return err
	}

	ctx.Artifacts.Add(a)
	return nil
}
