// Package build provides a pipe that can build binaries for several
// languages.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/deprecate"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/internal/skips"
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
	g := semerrgroup.New(ctx.Parallelism)
	for _, build := range ctx.Config.Builds {
		if build.Skip {
			log.WithField("id", build.ID).Info("skip is set")
			continue
		}
		log.WithField("build", build).Debug("building")
		runPipeOnBuild(ctx, g, build)
	}
	return g.Wait()
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	if !reflect.DeepEqual(ctx.Config.SingleBuild, config.Build{}) {
		deprecate.Notice(ctx, "build")
	}

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

func runPipeOnBuild(ctx *context.Context, g semerrgroup.Group, build config.Build) {
	for _, target := range filter(ctx, build.Targets) {
		g.Go(func() error {
			opts, err := buildOptionsForTarget(ctx, build, target)
			if err != nil {
				return err
			}

			if !skips.Any(ctx, skips.PreBuildHooks) {
				if err := runHook(ctx, *opts, build.Env, build.Hooks.Pre); err != nil {
					return fmt.Errorf("pre hook failed: %w", err)
				}
			}
			if err := doBuild(ctx, build, *opts); err != nil {
				return err
			}
			if !skips.Any(ctx, skips.PostBuildHooks) {
				if err := runHook(ctx, *opts, build.Env, build.Hooks.Post); err != nil {
					return fmt.Errorf("post hook failed: %w", err)
				}
			}
			return nil
		})
	}
}

func runHook(ctx *context.Context, opts builders.Options, buildEnv []string, hooks config.Hooks) error {
	if len(hooks) == 0 {
		return nil
	}

	for _, hook := range hooks {
		var env []string

		env = append(env, ctx.Env.Strings()...)
		for _, rawEnv := range append(buildEnv, hook.Env...) {
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

		if err := shell.Run(ctx, dir, cmd, env, hook.Output); err != nil {
			return err
		}
	}

	return nil
}

func doBuild(ctx *context.Context, build config.Build, opts builders.Options) error {
	return builders.For(build.Builder).Build(ctx, build, opts)
}

func buildOptionsForTarget(ctx *context.Context, build config.Build, target string) (*builders.Options, error) {
	ext := extFor(target, build.BuildDetails)
	parts := strings.Split(target, "_")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	goos := parts[0]
	goarch := parts[1]

	var gomips string
	var goarm string
	var goamd64 string
	if strings.HasPrefix(goarch, "arm") && len(parts) > 2 {
		goarm = parts[2]
	}
	if strings.HasPrefix(goarch, "mips") && len(parts) > 2 {
		gomips = parts[2]
	}
	if strings.HasPrefix(goarch, "amd64") && len(parts) > 2 {
		goamd64 = parts[2]
	}

	buildOpts := builders.Options{
		Target:  target,
		Ext:     ext,
		Goos:    goos,
		Goarch:  goarch,
		Goarm:   goarm,
		Gomips:  gomips,
		Goamd64: goamd64,
	}

	bin, err := tmpl.New(ctx).WithBuildOptions(buildOpts).Apply(build.Binary)
	if err != nil {
		return nil, err
	}

	name := bin + ext
	dir := fmt.Sprintf("%s_%s", build.ID, target)
	if build.NoUniqueDistDir {
		dir = ""
	}
	relpath := filepath.Join(ctx.Config.Dist, dir, name)
	path, err := filepath.Abs(relpath)
	if err != nil {
		return nil, err
	}
	buildOpts.Path = path
	buildOpts.Name = name

	log.WithField("binary", relpath).Info("building")
	return &buildOpts, nil
}

func extFor(target string, build config.BuildDetails) string {
	// Configure the extensions for shared and static libraries - by default .so and .a respectively -
	// with overrides for Windows (.dll for shared and .lib for static) and .dylib for macOS.
	switch build.Buildmode {
	case "c-shared":
		if strings.Contains(target, "darwin") {
			return ".dylib"
		}
		if strings.Contains(target, "windows") {
			return ".dll"
		}
		return ".so"
	case "c-archive":
		if strings.Contains(target, "windows") {
			return ".lib"
		}
		return ".a"
	}

	if target == "js_wasm" {
		return ".wasm"
	}

	if strings.Contains(target, "windows") {
		return ".exe"
	}

	return ""
}
