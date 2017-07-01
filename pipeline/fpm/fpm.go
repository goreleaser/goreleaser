// Package fpm implements the Pipe interface providing FPM bindings.
package fpm

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/apex/log"
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
		log.Info("no output formats configured, skipping")
		return nil
	}
	_, err := exec.LookPath("fpm")
	if err != nil {
		return ErrNoFPM
	}

	var g errgroup.Group
	for _, format := range ctx.Config.FPM.Formats {
		for _, build := range ctx.Config.Builds {
			for _, goarch := range build.Goarch {
				var key = build.Binary + "linux" + goarch
				// TODO: fix here
				if ctx.Binaries[key] == "" {
					continue
				}
				format := format
				// TODO: fix here
				archive := ctx.Binaries[key]
				arch := goarchToUnix[goarch]
				g.Go(func() error {
					return create(ctx, format, archive, arch)
				})
			}
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, format, archive, arch string) error {
	var path = filepath.Join(ctx.Config.Dist, archive)
	var file = path + "." + format
	log.WithField("file", file).Info("Creating")

	var options = []string{
		"--input-type", "dir",
		"--output-type", format,
		"--name", ctx.Config.Name,
		"--version", ctx.Version,
		"--architecture", arch,
		"--chdir", path,
		"--package", file,
		"--force",
	}

	if ctx.Config.FPM.Vendor != "" {
		options = append(options, "--vendor", ctx.Config.FPM.Vendor)
	}
	if ctx.Config.FPM.Homepage != "" {
		options = append(options, "--url", ctx.Config.FPM.Homepage)
	}
	if ctx.Config.FPM.Maintainer != "" {
		options = append(options, "--maintainer", ctx.Config.FPM.Maintainer)
	}
	if ctx.Config.FPM.Description != "" {
		options = append(options, "--description", ctx.Config.FPM.Description)
	}
	if ctx.Config.FPM.License != "" {
		options = append(options, "--license", ctx.Config.FPM.License)
	}
	for _, dep := range ctx.Config.FPM.Dependencies {
		options = append(options, "--depends", dep)
	}
	for _, conflict := range ctx.Config.FPM.Conflicts {
		options = append(options, "--conflicts", conflict)
	}

	for _, build := range ctx.Config.Builds {
		// This basically tells fpm to put the binary in the /usr/local/bin
		// binary=/usr/local/bin/binary
		options = append(options, fmt.Sprintf(
			"%s=%s",
			build.Binary,
			filepath.Join("/usr/local/bin", build.Binary),
		))
	}

	if out, err := exec.Command("fpm", options...).CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	ctx.AddArtifact(file)
	return nil
}
