package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
	"github.com/goreleaser/goreleaser/internal/ext"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Pipe for build
type Pipe struct{}

func (Pipe) String() string {
	return "building binaries"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	for _, build := range ctx.Config.Builds {
		log.WithField("build", build).Debug("building")
		if err := checkMain(ctx, build); err != nil {
			return err
		}
		if err := runPipeOnBuild(ctx, build); err != nil {
			return err
		}
	}
	return nil
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	for i, build := range ctx.Config.Builds {
		ctx.Config.Builds[i] = buildWithDefaults(ctx, build)
	}
	if len(ctx.Config.Builds) == 0 {
		ctx.Config.Builds = []config.Build{
			buildWithDefaults(ctx, ctx.Config.SingleBuild),
		}
	}
	return nil
}

func buildWithDefaults(ctx *context.Context, build config.Build) config.Build {
	if build.Binary == "" {
		build.Binary = ctx.Config.Release.GitHub.Name
	}
	if build.Main == "" {
		build.Main = "."
	}
	if len(build.Goos) == 0 {
		build.Goos = []string{"linux", "darwin"}
	}
	if len(build.Goarch) == 0 {
		build.Goarch = []string{"amd64", "386"}
	}
	if len(build.Goarm) == 0 {
		build.Goarm = []string{"6"}
	}
	if build.Ldflags == "" {
		build.Ldflags = "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"
	}
	return build
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	if err := runHook(build.Env, build.Hooks.Pre); err != nil {
		return errors.Wrap(err, "pre hook failed")
	}
	sem := make(chan bool, ctx.Parallelism)
	var g errgroup.Group
	for _, target := range buildtarget.All(build) {
		sem <- true
		target := target
		build := build
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return doBuild(ctx, build, target)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return errors.Wrap(runHook(build.Env, build.Hooks.Post), "post hook failed")
}

func runHook(env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("running hook")
	cmd := strings.Fields(hook)
	return run(buildtarget.Runtime, cmd, env)
}

func doBuild(ctx *context.Context, build config.Build, target buildtarget.Target) error {
	var binaryName = build.Binary + ext.For(target)
	var prettyName = binaryName
	if ctx.Config.Archive.Format == "binary" {
		var err error
		binaryName, err = nameFor(ctx, target, build.Binary)
		if err != nil {
			return err
		}
		binaryName = binaryName + ext.For(target)
	}
	folder, err := nameFor(ctx, target, ctx.Config.ProjectName)
	if err != nil {
		return err
	}
	var binary = filepath.Join(ctx.Config.Dist, folder, binaryName)
	ctx.AddBinary(target.String(), folder, prettyName, binary)
	log.WithField("binary", binary).Info("building")
	cmd := []string{"go", "build"}
	if build.Flags != "" {
		cmd = append(cmd, strings.Fields(build.Flags)...)
	}
	flags, err := ldflags(ctx, build)
	if err != nil {
		return err
	}
	cmd = append(cmd, "-ldflags="+flags, "-o", binary, build.Main)
	return errors.Wrapf(run(target, cmd, build.Env), "failed to build for %s", target)
}

func run(target buildtarget.Target, command, env []string) error {
	/* #nosec */
	var cmd = exec.Command(command[0], command[1:]...)
	env = append(env, target.Env()...)
	var log = log.WithField("target", target.PrettyString()).
		WithField("env", env).
		WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return errors.New(string(out))
	}
	return nil
}
