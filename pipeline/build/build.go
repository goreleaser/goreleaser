package build

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goreleaser/goreleaser/context"
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
	if ctx.Config.Build.Hooks.Pre != "" {
		log.Println("Running pre-build hook", ctx.Config.Build.Hooks.Pre)
		cmd := strings.Fields(ctx.Config.Build.Hooks.Pre)
		if err := run(runtime.GOOS, runtime.GOARCH, cmd); err != nil {
			return err
		}
	}
	var g errgroup.Group
	for _, goos := range ctx.Config.Build.Goos {
		for _, goarch := range ctx.Config.Build.Goarch {
			goos := goos
			goarch := goarch
			if !valid(goos, goarch) {
				continue
			}
			name, err := nameFor(ctx, goos, goarch)
			if err != nil {
				return err
			}
			ctx.Archives[goos+goarch] = name
			g.Go(func() error {
				return build(name, goos, goarch, ctx)
			})
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if ctx.Config.Build.Hooks.Post != "" {
		log.Println("Running post-build hook", ctx.Config.Build.Hooks.Post)
		cmd := strings.Fields(ctx.Config.Build.Hooks.Post)
		if err := run(runtime.GOOS, runtime.GOARCH, cmd); err != nil {
			return err
		}
	}
	return nil
}

func build(name, goos, goarch string, ctx *context.Context) error {
	output := filepath.Join(
		ctx.Config.Dist,
		name,
		ctx.Config.Build.Binary+extFor(goos),
	)
	log.Println("Building", output)
	cmd := []string{"go", "build"}
	if ctx.Config.Build.Flags != "" {
		cmd = append(cmd, strings.Fields(ctx.Config.Build.Flags)...)
	}
	flags, err := ldflags(ctx)
	if err != nil {
		return err
	}
	log.Println(flags)
	cmd = append(cmd, "-ldflags="+flags, "-o", output, ctx.Config.Build.Main)
	if err := run(goos, goarch, cmd); err != nil {
		return err
	}
	return nil
}

func run(goos, goarch string, command []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "GOOS="+goos, "GOARCH="+goarch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	return nil
}
