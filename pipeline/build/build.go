package build

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/goreleaser/releaser/context"
	"golang.org/x/sync/errgroup"
)

// Pipe for build
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Build"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	for _, system := range ctx.Config.Build.Oses {
		for _, arch := range ctx.Config.Build.Arches {
			system := system
			arch := arch
			name, err := ctx.ArchiveName(system, arch)
			if err != nil {
				return err
			}
			ctx.Archives = append(ctx.Archives, name)
			g.Go(func() error {
				return build(name, system, arch, ctx)
			})
		}
	}
	return g.Wait()
}

func build(name, system, arch string, ctx *context.Context) error {
	ldflags := ctx.Config.Build.Ldflags + " -X main.version=" + ctx.Git.CurrentTag
	output := "dist/" + name + "/" + ctx.Config.BinaryName + extFor(system)
	log.Println("Building", output, "...")
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags="+ldflags,
		"-o", output,
		ctx.Config.Build.Main,
	)
	cmd.Env = append(
		cmd.Env,
		"GOOS="+system,
		"GOARCH="+arch,
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

func extFor(system string) string {
	if system == "windows" {
		return ".exe"
	}
	return ""
}
