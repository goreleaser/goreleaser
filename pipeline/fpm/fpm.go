// Package fpm implements the Pipe interface providing FPM bindings.
package fpm

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/pkg/errors"
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
	return doRun(ctx)
}

func doRun(ctx *context.Context) error {
	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, format := range ctx.Config.FPM.Formats {
		for platform, groups := range ctx.Binaries {
			if !strings.Contains(platform, "linux") {
				log.WithField("platform", platform).Debug("skipped non-linux builds for fpm")
				continue
			}
			sem <- true
			format := format
			arch := linux.Arch(platform)
			for folder, binaries := range groups {
				g.Go(func() error {
					defer func() {
						<-sem
					}()
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
	dir, err := ioutil.TempDir("", "fpm")
	if err != nil {
		return err
	}
	log.WithField("file", file).WithField("workdir", dir).Info("creating fpm archive")
	var options = basicOptions(ctx, dir, format, arch, file)

	for _, binary := range binaries {
		// This basically tells fpm to put the binary in the bindir, e.g. /usr/local/bin
		// binary=/usr/local/bin/binary
		log.WithField("path", binary.Path).
			WithField("name", binary.Name).
			Debug("added binary to fpm package")
		options = append(options, fmt.Sprintf(
			"%s=%s",
			binary.Path,
			filepath.Join(ctx.Config.FPM.Bindir, binary.Name),
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
		return errors.Wrap(err, string(out))
	}
	ctx.AddArtifact(file)
	return nil
}

func basicOptions(ctx *context.Context, workdir, format, arch, file string) []string {
	var options = []string{
		"--input-type", "dir",
		"--output-type", format,
		"--name", ctx.Config.ProjectName,
		"--version", ctx.Version,
		"--architecture", arch,
		"--package", file,
		"--force",
		"--workdir", workdir,
	}

	if ctx.Debug {
		options = append(options, "--debug")
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
	return options
}
