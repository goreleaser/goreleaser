package build

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/uname"
	"golang.org/x/sync/errgroup"
)

// Pipe for build
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Build"
}

// Run the pipe
func (Pipe) Run(config config.ProjectConfig) error {
	var g errgroup.Group
	for _, system := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			system := system
			arch := arch
			g.Go(func() error {
				return build(system, arch, config)
			})
		}
	}
	return g.Wait()
}

func build(system, arch string, config config.ProjectConfig) error {
	log.Println("Building", system+"/"+arch, "...")
	name, err := config.NameFor(system, arch, config.Git.CurrentTag)
	if err != nil {
		return err
	}
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags=-s -w -X main.version="+config.Git.CurrentTag,
		"-o", "dist/"+name+"/"+config.BinaryName,
		config.Build.Main,
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
