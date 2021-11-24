// Package build provides a pipe that can build binaries for several
// languages.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/caarlos0/go-shellwords"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	builders "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"

	// langs to init.
	_ "github.com/goreleaser/goreleaser/internal/builders/golang"
)

// Pipe for build.
type Pipe struct{}

func (Pipe) String() string {
	return "building binaries"
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, build := range ctx.Config.Builds {
		if build.Skip {
			log.WithField("id", build.ID).Info("skip is set")
			continue
		}
		log.WithField("build", build).Debug("building")
		if err := runPipeOnBuild(ctx, build); err != nil {
			return err
		}
	}
	return nil
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("builds")
	for i, build := range ctx.Config.Builds {
		build, err := buildWithDefaults(ctx, build)
		if err != nil {
			return err
		}
		ctx.Config.Builds[i] = build
		ids.Inc(ctx.Config.Builds[i].ID)
	}
	if len(ctx.Config.Builds) == 0 {
		build, err := buildWithDefaults(ctx, ctx.Config.SingleBuild)
		if err != nil {
			return err
		}
		ctx.Config.Builds = []config.Build{build}
	}
	return ids.Validate()
}

func buildWithDefaults(ctx *context.Context, build config.Build) (config.Build, error) {
	if build.Builder == "" {
		build.Builder = "go"
	}
	if build.Binary == "" {
		build.Binary = ctx.Config.ProjectName
	}
	if build.ID == "" {
		build.ID = ctx.Config.ProjectName
	}
	for k, v := range build.Env {
		build.Env[k] = os.ExpandEnv(v)
	}
	return builders.For(build.Builder).WithDefaults(build)
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	g := semerrgroup.New(ctx.Parallelism)
	for _, target := range build.Targets {
		target := target
		build := build
		g.Go(func() error {
			opts, err := buildOptionsForTarget(ctx, build, target)
			if err != nil {
				return err
			}

			if err := runHook(ctx, *opts, build.Env, build.Hooks.Pre); err != nil {
				return fmt.Errorf("pre hook failed: %w", err)
			}
			if err := doBuild(ctx, build, *opts); err != nil {
				return err
			}
			if !ctx.SkipPostBuildHooks {
				if err := runHook(ctx, *opts, build.Env, build.Hooks.Post); err != nil {
					return fmt.Errorf("post hook failed: %w", err)
				}
			}
			return nil
		})
	}

	return g.Wait()
}

func runHook(ctx *context.Context, opts builders.Options, buildEnv []string, hooks config.Hooks) error {
	if len(hooks) == 0 {
		return nil
	}

	for _, hook := range hooks {
		var env []string

		env = append(env, ctx.Env.Strings()...)
		env = append(env, buildEnv...)

		for _, rawEnv := range hook.Env {
			e, err := tmpl.New(ctx).WithBuildOptions(opts).Apply(rawEnv)
			if err != nil {
				return err
			}
			env = append(env, e)
		}

		dir, err := tmpl.New(ctx).WithBuildOptions(opts).Apply(hook.Dir)
		if err != nil {
			return err
		}

		sh, err := tmpl.New(ctx).WithBuildOptions(opts).
			WithEnvS(env).
			Apply(hook.Cmd)
		if err != nil {
			return err
		}

		log.WithField("hook", sh).Info("running hook")
		cmd, err := shellwords.Parse(sh)
		if err != nil {
			return err
		}

		if err := shell.Run(ctx, dir, cmd, env); err != nil {
			return err
		}
	}

	return nil
}

func doBuild(ctx *context.Context, build config.Build, opts builders.Options) error {
	return builders.For(build.Builder).Build(ctx, build, opts)
}

func buildOptionsForTarget(ctx *context.Context, build config.Build, target string) (*builders.Options, error) {
	ext := extFor(target, build.Flags)
	parts := strings.Split(target, "_")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	goos := parts[0]
	goarch := parts[1]

	var gomips string
	var goarm string
	if strings.HasPrefix(goarch, "arm") && len(parts) > 2 {
		goarm = parts[2]
	}
	if strings.HasPrefix(goarch, "mips") && len(parts) > 2 {
		gomips = parts[2]
	}

	buildOpts := builders.Options{
		Target: target,
		Ext:    ext,
		Goos:   goos,
		Goarch: goarch,
		Goarm:  goarm,
		Gomips: gomips,
	}

	binary, err := tmpl.New(ctx).WithBuildOptions(buildOpts).Apply(build.Binary)
	if err != nil {
		return nil, err
	}

	build.Binary = binary
	name := build.Binary + ext
	dir := fmt.Sprintf("%s_%s", build.ID, target)
	if build.NoUniqueDistDir {
		dir = ""
	}
	path, err := filepath.Abs(filepath.Join(ctx.Config.Dist, dir, name))
	if err != nil {
		return nil, err
	}
	buildOpts.Path = path
	buildOpts.Name = name

	log.WithField("binary", buildOpts.Path).Info("building")
	return &buildOpts, nil
}

func extFor(target string, flags config.FlagArray) string {
	if strings.Contains(target, "windows") {
		for _, s := range flags {
			if s == "-buildmode=c-shared" {
				return ".dll"
			}
			if s == "-buildmode=c-archive" {
				return ".lib"
			}
		}
		return ".exe"
	}
	if target == "js_wasm" {
		return ".wasm"
	}
	return ""
}
