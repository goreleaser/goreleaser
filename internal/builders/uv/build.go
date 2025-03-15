package uv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/common"
	"github.com/goreleaser/goreleaser/v2/internal/pyproject"
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

const defaultTarget = "none-any"

//nolint:gochecknoinits
func init() {
	api.Register("uv", Default)
}

// Builder is golang builder.
type Builder struct{}

// Dependencies implements build.DependingBuilder.
func (b *Builder) Dependencies() []string {
	return []string{"uv"}
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	if target != defaultTarget {
		log.Warn("uv only accepts the target 'none-any'")
	}
	return Target{}, nil
}

// Target is the UV build target.
type Target struct{}

// Fields implements build.Target.
func (Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   "all",
		tmpl.KeyArch: "all",
	}
}

// String implements fmt.Stringer.
func (Target) String() string {
	return defaultTarget
}

var once sync.Once

// WithDefaults implements build.Builder.
func (b *Builder) WithDefaults(build config.Build) (config.Build, error) {
	once.Do(func() {
		log.Warn("you are using the experimental UV builder")
	})

	if len(build.Targets) == 0 {
		build.Targets = []string{defaultTarget}
	}

	if build.Tool == "" {
		build.Tool = "uv"
	}

	if build.Command == "" {
		build.Command = "build"
	}

	if build.Dir == "" {
		build.Dir = "."
	}

	if build.Main != "" {
		return build, errors.New("main is not used for uv")
	}

	proj, err := pyproject.Open(filepath.Join(build.Dir, "pyproject.toml"))
	if err != nil {
		return build, fmt.Errorf("uv: could not open pyproject.toml: %w", err)
	}

	if build.Buildmode == "" {
		build.Buildmode = "wheel"
	}

	name := strings.ReplaceAll(proj.Project.Name, "-", "_")
	switch build.Buildmode {
	case "wheel":
		build.Binary = fmt.Sprintf("%s-%s-py3-none-any", name, proj.Project.Version)
	case "sdist":
		build.Binary = fmt.Sprintf("%s-%s", name, proj.Project.Version)
	}

	if err := common.ValidateNonGoConfig(build, common.WithBuildMode); err != nil {
		return build, err
	}

	return build, nil
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	var (
		atype      = artifact.PyWheel
		buildFlags = []string{"--wheel"}
	)
	if build.Buildmode == "sdist" {
		atype = artifact.PySdist
		buildFlags = []string{"--sdist"}
	}

	a := &artifact.Artifact{
		Type:   atype,
		Path:   options.Path,
		Name:   options.Name,
		Goos:   "all",
		Goarch: "all",
		Target: options.Target.String(),
		Extra: map[string]interface{}{
			artifact.ExtraExt:     options.Ext,
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "uv",
		},
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithBuildOptions(options).
		WithEnvS(env).
		WithArtifact(a)

	uvbin, err := tpl.Apply(build.Tool)
	if err != nil {
		return err
	}

	command := []string{
		uvbin,
		build.Command,
		"--out-dir", filepath.Dir(options.Path),
	}
	command = append(command, buildFlags...)

	tenv, err := common.TemplateEnv(build.Env, tpl)
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
