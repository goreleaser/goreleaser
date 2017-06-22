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
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/ext"
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
	if err := runHook(ctx.Config.Build.Env, ctx.Config.Build.Hooks.Pre); err != nil {
		return err
	}
	sem := make(chan bool, 4)
	var g errgroup.Group
	for _, target := range buildTargets(ctx) {
		name, err := nameFor(ctx, target)
		if err != nil {
			return err
		}
		ctx.Archives[target.String()] = name

		sem <- true
		target := target
		g.Go(func() error {
			defer func() {
				<-sem
			}()
			return build(ctx, name, target)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return runHook(ctx.Config.Build.Env, ctx.Config.Build.Hooks.Post)
}

func runHook(env []string, hook string) error {
	if hook == "" {
		return nil
	}
	log.WithField("hook", hook).Info("Running hook")
	cmd := strings.Fields(hook)
	return run(runtimeTarget, cmd, env)
}

func build(ctx *context.Context, name string, target buildTarget) error {
	output := filepath.Join(
		ctx.Config.Dist,
		name,
		ctx.Config.Build.Binary+ext.For(target.goos),
	)
	log.WithField("binary", output).Info("Building")
	cmd := []string{"go", "build"}
	if ctx.Config.Build.Flags != "" {
		cmd = append(cmd, strings.Fields(ctx.Config.Build.Flags)...)
	}
	flags, err := ldflags(ctx)
	if err != nil {
		return err
	}
	cmd = append(cmd, "-ldflags="+flags, "-o", output, ctx.Config.Build.Main)
	return run(target, cmd, ctx.Config.Build.Env)
}

func run(target buildTarget, command, env []string) error {
	var cmd = exec.Command(command[0], command[1:]...)
	env = append(env, "GOOS="+target.goos, "GOARCH="+target.goarch, "GOARM="+target.goarm)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, env...)
	log.WithField("target", target.PrettyString()).
		WithField("env", env).
		WithField("args", cmd.Args).
		Debug("Running")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build failed: %s\n%v", target.PrettyString(), string(out))
	}
	return nil
}
