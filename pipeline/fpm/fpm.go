// Package fpm implements the Pipe interface providing FPM bindings.
package fpm

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/pipeline"
	"golang.org/x/sync/errgroup"
)

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
		return pipeline.Skip("no output formats configured")
	}
	_, err := exec.LookPath("fpm")
	if err != nil {
		return ErrNoFPM
	}

	var g errgroup.Group
	for _, format := range ctx.Config.FPM.Formats {
		for platform, groups := range ctx.Binaries {
			if !strings.Contains(platform, "linux") {
				log.WithField("platform", platform).Debug("skipped non-linux builds for fpm")
				continue
			}
			format := format
			arch := linux.Arch(platform)
			for folder, binaries := range groups {
				g.Go(func() error {
					return create(ctx, format, folder, arch, binaries)
				})
			}
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, format, folder, arch string, binaries []context.Binary) error {
	var path = filepath.Join(ctx.Config.Dist, folder)
	var file = path + "." + format
	var log = log.WithField("format", format).WithField("arch", arch)
	log.WithField("file", file).Info("creating fpm archive")

	var options = []string{
		"--input-type", "dir",
		"--output-type", format,
		"--name", ctx.Config.ProjectName,
		"--version", ctx.Version,
		"--architecture", arch,
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

	// FPM requires --rpm-os=linux if your rpm target is linux
	if format == "rpm" {
		options = append(options, "--rpm-os", "linux")
	}

	for _, binary := range binaries {
		// This basically tells fpm to put the binary in the /usr/local/bin
		// binary=/usr/local/bin/binary
		log.WithField("path", binary.Path).
			WithField("name", binary.Name).
			Debug("added binary to fpm package")
		options = append(options, fmt.Sprintf(
			"%s=%s",
			binary.Path,
			filepath.Join("/usr/local/bin", binary.Name),
		))
	}

	for src, dest := range ctx.Config.FPM.Files {
		log.WithField("src", src).
			WithField("dest", dest).
			Debug("added an extra file to the fpm package")
		options = append(options, fmt.Sprintf(
			"%s=%s",
			src,
			dest,
		))
	}

	log.WithField("args", options).Debug("creating fpm package")
	if out, err := exec.Command("fpm", options...).CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	ctx.AddArtifact(file)
	return nil
}
