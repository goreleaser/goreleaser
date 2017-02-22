package build

import (
	"errors"
	"log"
	"os"
	"os/exec"
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
	var g errgroup.Group
	for _, goos := range ctx.Config.Build.Goos {
		for _, goarch := range ctx.Config.Build.Goarch {
			goos := goos
			goarch := goarch
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
	return g.Wait()
}

func build(name, goos, goarch string, ctx *context.Context) error {
	ldflags := ctx.Config.Build.Ldflags + " -X main.version=" + ctx.Version
	output := "dist/" + name + "/" + ctx.Config.Build.BinaryName + extFor(goos)
	log.Println("Building", output)
	if ctx.Config.Build.Hooks.Pre != "" {
		cmd := strings.Split(ctx.Config.Build.Hooks.Pre, " ")
		if err := run(goos, goarch, cmd); err != nil {
			return err
		}
	}
	cmd := []string{"go", "build"}
	if ctx.Config.Build.Flags != "" {
		cmd = append(cmd, strings.Split(ctx.Config.Build.Flags, " ")...)
	}
	cmd = append(cmd, "-ldflags="+ldflags, "-o", output, ctx.Config.Build.Main)
	if err := run(goos, goarch, cmd); err != nil {
		return err
	}
	if ctx.Config.Build.Hooks.Post != "" {
		cmd := strings.Split(ctx.Config.Build.Hooks.Post, " ")
		if err := run(goos, goarch, cmd); err != nil {
			return err
		}
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
