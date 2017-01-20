package build

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"

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
	ldflags := ctx.Config.Build.Ldflags + " -X main.version=" + ctx.Git.CurrentTag
	output := "dist/" + name + "/" + ctx.Config.Build.BinaryName + extFor(goos)
	log.Println("Building", output)
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags="+ldflags,
		"-o", output,
		ctx.Config.Build.Main,
	)
	cmd.Env = append(
		cmd.Env,
		"GOOS="+goos,
		"GOARCH="+goarch,
		"GOROOT="+os.Getenv("GOROOT"),
		"GOPATH="+os.Getenv("GOPATH"),
	)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return errors.New(stdout.String())
	}
	return nil
}
