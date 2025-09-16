// Package build provides a pipe that can build binaries for several
// languages.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/go-shellwords"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/v2/internal/shell"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	builders "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"

	// langs to init.
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/bun"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/deno"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/golang"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/poetry"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/rust"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/uv"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/zig"
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
		skip, err := tmpl.New(ctx).Bool(build.Skip)
		if err != nil {
			return err
		}
		if skip {
			log.WithField("id", build.ID).Info("skip is set")
			continue
		}
		log.WithField("build", build).Debug("building")
		if err := prepare(ctx, build); err != nil {
			return err
		}
		if allowParallelism(build) {
			runPipeOnBuild(ctx, g, build)
			continue
		}
		g.Go(func() error {
			gg := semerrgroup.New(1)
			runPipeOnBuild(ctx, gg, build)
			return gg.Wait()
		})
	}
	return g.Wait()
}

func allowParallelism(build config.Build) bool {
	conc, ok := builders.For(build.Builder).(builders.ConcurrentBuilder)
	if !ok {
		// assume concurrent
		return true
	}
	return conc.AllowConcurrentBuilds()
}

func prepare(ctx *context.Context, build config.Build) error {
	prep, ok := builders.For(build.Builder).(builders.PreparedBuilder)
	if !ok {
		// nothing to do
		return nil
	}
	return prep.Prepare(ctx, build)
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("builds")
	for i, build := range ctx.Config.Builds {
		if build.GoBinary != "" {
			build.Tool = build.GoBinary
			deprecate.Notice(ctx, "builds.gobinary")
		}
		build, err := buildWithDefaults(ctx, build)
		if err != nil {
			return err
		}
		ctx.Config.Builds[i] = build
		ids.Inc(ctx.Config.Builds[i].ID)
	}
	if len(ctx.Config.Builds) == 0 {
		build, err := buildWithDefaults(ctx, config.Build{})
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
		build.InternalDefaults.Binary = true
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
	for _, target := range filter(ctx, build) {
		g.Go(func() error {
			if err := buildTarget(ctx, build, target); err != nil {
				return gerrors.Wrap(err, "", "target", target)
			}
			return nil
		})
	}
}

func buildTarget(ctx *context.Context, build config.Build, target string) error {
	opts, err := buildOptionsForTarget(ctx, build, target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	if !skips.Any(ctx, skips.PreBuildHooks) {
		if err := runHook(ctx, *opts, build.Env, build.Hooks.Pre); err != nil {
			return fmt.Errorf("pre hook failed: %w", err)
		}
	}

	if err := doBuild(ctx, build, *opts); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if !skips.Any(ctx, skips.PostBuildHooks) {
		if err := runHook(ctx, *opts, build.Env, build.Hooks.Post); err != nil {
			return fmt.Errorf("post hook failed: %w", err)
		}
	}

	return nil
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
	buildOpts := builders.Options{
		Ext: ext,
	}

	t, err := builders.For(build.Builder).Parse(target)
	if err != nil {
		return nil, err
	}
	buildOpts.Target = t

	bin, err := tmpl.New(ctx).WithBuildOptions(buildOpts).Apply(build.Binary)
	if err != nil {
		return nil, err
	}

	name := bin + ext
	dir := fmt.Sprintf("%s_%s", build.ID, t)
	noUnique, err := tmpl.New(ctx).Bool(build.NoUniqueDistDir)
	if err != nil {
		return nil, err
	}
	if noUnique {
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

// TODO: this should probably be the responsibility of each builder.
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
		if strings.Contains(target, "wasm") {
			return ".wasm"
		}
		return ".so"
	case "c-archive":
		if strings.Contains(target, "windows") {
			return ".lib"
		}
		return ".a"
	}

	if strings.Contains(target, "wasm") {
		return ".wasm"
	}

	if strings.Contains(target, "windows") {
		return ".exe"
	}

	return ""
}
