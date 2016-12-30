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

type Pipe struct{}

func (Pipe) Name() string {
	return "Build"
}

func (Pipe) Work(config config.ProjectConfig) error {
	var g errgroup.Group
	for _, bos := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			bos := bos
			arch := arch
			g.Go(func() error {
				return build(bos, arch, config)
			})
		}
	}
	return g.Wait()
}

func build(bos, arch string, config config.ProjectConfig) error {
	log.Println("Building", bos+"/"+arch, "...")
	cmd := exec.Command(
		"go",
		"build",
		"-ldflags=-s -w -X main.version="+config.Git.CurrentTag,
		"-o", target(bos, arch, config.BinaryName),
		config.Build.Main,
	)
	cmd.Env = append(
		cmd.Env,
		"GOOS="+bos,
		"GOARCH="+arch,
		"GOROOT="+os.Getenv("GOROOT"),
		"GOPATH="+os.Getenv("GOPATH"),
	)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err := cmd.Run()
	if err != nil {
		return errors.New(stdout.String())
	}
	return nil
}

func target(os, arch, binary string) string {
	return "dist/" + binary + "_" + uname.FromGo(os) + "_" + uname.FromGo(arch) + "/" + binary
}
