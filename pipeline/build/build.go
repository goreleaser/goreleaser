// Package build implements Pipe and can build Go projects for
// several platforms, with pre and post hook support.
package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/ext"
	"github.com/goreleaser/goreleaser/internal/name"
	"golang.org/x/sync/errgroup"
)

// Pipe for build
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Building binaries"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	for _, build := range ctx.Config.Builds {
		log.WithField("build", build).Debug("building")
		if err := runPipeOnBuild(ctx, build); err != nil {
			return err
		}
	}
	return nil
}

func runPipeOnBuild(ctx *context.Context, build config.Build) error {
	if err := runHook(build.Env, build.Hooks.Pre); err != nil {
		return err
	}
	sem := make(chan bool, 4)
	var g errgroup.Group
	for _, target := range buildTargets(build) {
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
	return runHook(build.Env, build.Hooks.Post)
}

func runHook(env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("running hook")
	cmd := strings.Fields(hook)
	return run(runtimeTarget, cmd, env)
}

func doBuild(ctx *context.Context, build config.Build, target buildTarget) error {
	folder, err := name.For(ctx, target.goos, target.goarch, target.goarm)
	if err != nil {
		return err
	}
	ctx.AddFolder(target.String(), folder)
	var binary = filepath.Join(
		ctx.Config.Dist,
		folder,
		build.Binary+ext.For(target.goos),
	)
	if ctx.Config.Archive.Format == "binary" {
		bin, err := name.ForBuild(ctx, build, target.goos, target.goarch, target.goarm)
		if err != nil {
			return err
		}
		binary = filepath.Join(
			ctx.Config.Dist,
			folder,
			bin+ext.For(target.goos),
		)
	}
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
	return run(target, cmd, build.Env)
}

func run(target buildTarget, command, env []string) error {
	var cmd = exec.Command(command[0], command[1:]...)
	env = append(env, "GOOS="+target.goos, "GOARCH="+target.goarch, "GOARM="+target.goarm)
	var log = log.WithField("target", target.PrettyString()).
		WithField("env", env).
		WithField("cmd", command)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.Debug("running")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).Debug("failed")
		return fmt.Errorf("build failed for %s:\n%v", target.PrettyString(), string(out))
	}
	return nil
}
