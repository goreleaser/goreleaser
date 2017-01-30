package fpm

import (
	"errors"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/sync/errgroup"
)

var goarchToUnix = map[string]string{
	"386":   "i386",
	"amd64": "x86_64",
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
	if len(ctx.Config.FPM.Formats) == 0 {
		log.Println("No output formats configured, skipping")
		return nil
	}
	_, err := exec.LookPath("fpm")
	if err != nil {
		return ErrNoFPM
	}

	var g errgroup.Group
	for _, format := range ctx.Config.FPM.Formats {
		for _, goarch := range ctx.Config.Build.Goarch {
			if ctx.Archives["linux"+goarch] == "" {
				continue
			}
			archive := ctx.Archives["linux"+goarch]
			arch := goarchToUnix[goarch]
			g.Go(func() error {
				return create(ctx, format, archive, arch)
			})
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, format, archive, arch string) error {
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
	for _, dep := range ctx.Config.FPM.Dependencies {
		options = append(options, "-d", dep)
	}
	// This basically tells fpm to put the binary in the /usr/local/bin
	// binary=/usr/local/bin/binary
	options = append(options, name+"="+filepath.Join("/usr/local/bin", name))
	cmd := exec.Command("fpm", options...)

	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	return nil
}
