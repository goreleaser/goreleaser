// Package uv provides Python builds using uv.
package uv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/builders/base"
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

	errSetBinary = errors.New("uv: binary name is set by uv itself")
	errTargets   = errors.New("uv: only target supported is 'none-any'")
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

	if !build.InternalDefaults.Binary && build.Binary != "" {
		return build, errSetBinary
	}

	if len(build.Targets) > 1 || build.Targets[0] != defaultTarget {
		return build, fmt.Errorf("%w: %s", errTargets, strings.Join(build.Targets, ","))
	}

	if err := base.ValidateNonGoConfig(build, base.WithBuildMode); err != nil {
		return build, err
	}

	return build, nil
}

// Build implements build.Builder.
func (b *Builder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	proj, err := pyproject.Open(filepath.Join(build.Dir, "pyproject.toml"))
	if err != nil {
		return fmt.Errorf("uv: could not open pyproject.toml: %w", err)
	}

	// options.Path will be dist/projectname-all-all/projectname.

	var buildFlags []string
	var art *artifact.Artifact
	switch build.Buildmode {
	case "wheel", "":
		buildFlags = []string{"--wheel"}
		art = wheel(proj, build, options)
	case "sdist":
		buildFlags = []string{"--sdist"}
		art = sdist(proj, build, options)
	default:
		return fmt.Errorf("uv: invalid buildmode %q", build.Buildmode)
	}

	env := []string{}
	env = append(env, ctx.Env.Strings()...)

	tpl := tmpl.New(ctx).
		WithEnvS(env)

	uvbin, err := tpl.Apply(build.Tool)
	if err != nil {
		return err
	}

	root := filepath.Dir(options.Path)

	command := []string{
		uvbin,
		build.Command,
		"--out-dir", root,
	}
	command = append(command, buildFlags...)

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

	if err := base.ChTimes(build, tpl, art); err != nil {
		return err
	}

	ctx.Artifacts.Add(art)
	return nil
}

func wheel(proj pyproject.PyProject, build config.Build, options api.Options) *artifact.Artifact {
	name := fmt.Sprintf("%s-%s-py3-none-any.whl", proj.Name(), proj.Project.Version)
	return &artifact.Artifact{
		Type:   artifact.PyWheel,
		Name:   name,
		Path:   filepath.Join(filepath.Dir(options.Path), name),
		Goos:   "all",
		Goarch: "all",
		Target: options.Target.String(),
		Extra: map[string]any{
			artifact.ExtraExt:     ".whl",
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "uv",
		},
	}
}

func sdist(proj pyproject.PyProject, build config.Build, options api.Options) *artifact.Artifact {
	name := fmt.Sprintf("%s-%s.tar.gz", proj.Name(), proj.Project.Version)
	return &artifact.Artifact{
		Type:   artifact.PySdist,
		Name:   name,
		Path:   filepath.Join(filepath.Dir(options.Path), name),
		Goos:   "all",
		Goarch: "all",
		Target: options.Target.String(),
		Extra: map[string]any{
			artifact.ExtraExt:     ".tar.gz",
			artifact.ExtraID:      build.ID,
			artifact.ExtraBuilder: "uv",
		},
	}
}
