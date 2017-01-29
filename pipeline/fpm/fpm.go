package fpm

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/goreleaser/context"
)

var linuxArchives = []struct {
	Key  string
	Arch string
}{
	{
		Key:  "linuxamd64",
		Arch: "x86_64",
	},
	{
		Key:  "linux386",
		Arch: "i386",
	},
}

// ErrNoFPM is shown when fpm cannot be found in $PATH
var ErrNoFPM = errors.New("fpm not present in $PATH")

// Pipe for fpm packaging
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating Linux packages with fpm"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	cmd := exec.Command("which", "fpm")
	if err := cmd.Run(); err != nil {
		return ErrNoFPM
	}
	if len(ctx.Config.FPM.Formats) == 0 {
		log.Println("No output formats configured")
		return nil
	}
	var g errgroup.Group
	for _, format := range ctx.Config.FPM.Formats {
		for _, archive := range linuxArchives {
			if ctx.Archives[archive.Key] == "" {
				continue
			}
			archive := archive
			g.Go(func() error {
				return create(
					ctx,
					format.Name,
					ctx.Archives[archive.Key],
					archive.Arch,
					format.Dependencies,
				)
			})
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, format, archive, arch string, deps []string) error {
	var path = filepath.Join("dist", archive)
	var file = path + ".deb"
	var name = ctx.Config.Build.BinaryName
	log.Println("Creating", file)

	var options = []string{
		"-s", "dir",
		"-t", format,
		"-n", name,
		"-v", ctx.Version,
		"-a", arch,
		"-C", path,
		"-p", file,
		"--force",
	}
	for _, dep := range deps {
		options = append(options, "-d", dep)
	}
	options = append(options, name+"="+filepath.Join("/usr/local/bin", name))
	cmd := exec.Command("fpm", options...)
	log.Println(cmd)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return errors.New(stdout.String())
	}
	return nil
}
