// Package build provides a pipe that can build binaries for several
// languages.
package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	builders "github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"

	// langs to init
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
	var ids = ids.New("builds")
	for i, build := range ctx.Config.Builds {
		ctx.Config.Builds[i] = buildWithDefaults(ctx, build)
		ids.Inc(ctx.Config.Builds[i].ID)
	}
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = []config.Build{
			buildWithDefaults(ctx, ctx.Config.SingleBuild),
		}
	}
	return ids.Validate()
}

func buildWithDefaults(ctx *context.Context, build config.Build) config.Build {
	if build.Lang == "" {
		build.Lang = "go"
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
	return builders.For(build.Lang).WithDefaults(build)
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	var g = semerrgroup.New(ctx.Parallelism)
	for _, target := range build.Targets {
		target := target
		build := build
		g.Go(func() error {
			opts, err := buildOptionsForTarget(ctx, build, target)
			if err != nil {
				return err
			}

			if err := runHook(ctx, *opts, build.Env, build.Hooks.Pre); err != nil {
				return errors.Wrap(err, "pre hook failed")
			}
			if err := doBuild(ctx, build, *opts); err != nil {
				return err
			}
			if !ctx.SkipPostBuildHooks {
				if err := runHook(ctx, *opts, build.Env, build.Hooks.Post); err != nil {
					return errors.Wrap(err, "post hook failed")
				}
			}
			return nil
		})
	}

	return g.Wait()
}

func runHook(ctx *context.Context, opts builders.Options, buildEnv []string, hooks config.BuildHooks) error {
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

		if err := run(ctx, dir, cmd, env); err != nil {
			return err
		}
	}

	return nil
}

func doBuild(ctx *context.Context, build config.Build, opts builders.Options) error {
	return builders.For(build.Lang).Build(ctx, build, opts)
}

func buildOptionsForTarget(ctx *context.Context, build config.Build, target string) (*builders.Options, error) {
	var ext = extFor(target, build.Flags)

	binary, err := tmpl.New(ctx).Apply(build.Binary)
	if err != nil {
		return nil, err
	}

	build.Binary = binary
	var name = build.Binary + ext
	path, err := filepath.Abs(
		filepath.Join(
			ctx.Config.Dist,
			fmt.Sprintf("%s_%s", build.ID, target),
			name,
		),
	)
	if err != nil {
		return nil, err
	}
	log.WithField("binary", path).Info("building")
	return &builders.Options{
		Target: target,
		Name:   name,
		Path:   path,
		Ext:    ext,
	}, nil
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

func run(ctx *context.Context, dir string, command, env []string) error {
	/* #nosec */
	var cmd = exec.CommandContext(ctx, command[0], command[1:]...)
	var log = log.WithField("env", env).WithField("cmd", command)
	cmd.Env = env
	if dir != "" {
		cmd.Dir = dir
	}
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.Wrapf(err, "%q", string(out))
	}
	return nil
}
