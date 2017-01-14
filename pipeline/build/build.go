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
func (Pipe) Run(context *context.Context) error {
	var g errgroup.Group
	for _, system := range context.Config.Build.Oses {
		for _, arch := range context.Config.Build.Arches {
			system := system
			arch := arch
			name, err := context.Config.ArchiveName(system, arch)
			if err != nil {
				return err
			}
			context.Archives = append(context.Archives, name)
			g.Go(func() error {
				return build(name, system, arch, context)
			})
		}
	}
	return g.Wait()
}

func build(name, system, arch string, context *context.Context) error {
	ldflags := context.Config.Build.Ldflags + " -X main.version=" + context.Git.CurrentTag
	output := "dist/" + name + "/" + context.Config.BinaryName
	log.Println("Building", output, "...")
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags="+ldflags,
		"-o", output,
		context.Config.Build.Main,
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
