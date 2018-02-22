// Package nfpm implements the Pipe interface providing NFPM bindings.
package nfpm

import (
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"

	"github.com/apex/log"
	"github.com/goreleaser/nfpm"
	"github.com/pkg/errors"

	// blank imports here because the formats implementations need register
	// themselves
	_ "github.com/goreleaser/nfpm/deb"
	_ "github.com/goreleaser/nfpm/rpm"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/filenametemplate"
	"github.com/goreleaser/goreleaser/internal/linux"
	"github.com/goreleaser/goreleaser/pipeline"
)

const defaultNameTemplate = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

// Pipe for fpm packaging
type Pipe struct{}

func (Pipe) String() string {
	return "creating Linux packages with nfpm"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var fpm = &ctx.Config.NFPM
	if fpm.Bindir == "" {
		fpm.Bindir = "/usr/local/bin"
	}
	if fpm.NameTemplate == "" {
		fpm.NameTemplate = defaultNameTemplate
	}
	if fpm.Files == nil {
		fpm.Files = make(map[string]string)
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if len(ctx.Config.NFPM.Formats) == 0 {
		return pipeline.Skip("no output formats configured")
	}
	return doRun(ctx)
}

func doRun(ctx *context.Context) error {
	var linuxBinaries = ctx.Artifacts.Filter(artifact.And(
		artifact.ByType(artifact.Binary),
		artifact.ByGoos("linux"),
	)).GroupByPlatform()
	var g errgroup.Group
	sem := make(chan bool, ctx.Parallelism)
	for _, format := range ctx.Config.NFPM.Formats {
		for platform, artifacts := range linuxBinaries {
			sem <- true
			format := format
			arch := linux.Arch(platform)
			artifacts := artifacts
			g.Go(func() error {
				defer func() {
					<-sem
				}()
				return create(ctx, format, arch, artifacts)
			})
		}
	}
	return g.Wait()
}

func create(ctx *context.Context, format, arch string, binaries []artifact.Artifact) error {
	name, err := filenametemplate.Apply(
		ctx.Config.NFPM.NameTemplate,
		filenametemplate.NewFields(ctx, ctx.Config.NFPM.Replacements, binaries...),
	)
	if err != nil {
		return err
	}
	var files = map[string]string{}
	for k, v := range ctx.Config.NFPM.Files {
		files[k] = v
	}
	var log = log.WithField("package", name+"."+format)
	for _, binary := range binaries {
		src := binary.Path
		dst := filepath.Join(ctx.Config.NFPM.Bindir, binary.Name)
		log.WithField("src", src).WithField("dst", dst).Debug("adding binary to package")
		files[src] = dst
	}
	log.WithField("files", files).Debug("all archive files")

	var info = nfpm.Info{
		Arch:        arch,
		Platform:    "linux",
		Conflicts:   ctx.Config.NFPM.Conflicts,
		Depends:     ctx.Config.NFPM.Dependencies,
		Recommends:  ctx.Config.NFPM.Recommends,
		Suggests:    ctx.Config.NFPM.Suggests,
		Name:        ctx.Config.ProjectName,
		Version:     ctx.Git.CurrentTag,
		Section:     "",
		Priority:    "",
		Maintainer:  ctx.Config.NFPM.Maintainer,
		Description: ctx.Config.NFPM.Description,
		Vendor:      ctx.Config.NFPM.Vendor,
		Homepage:    ctx.Config.NFPM.Homepage,
		License:     ctx.Config.NFPM.License,
		Bindir:      ctx.Config.NFPM.Bindir,
		Files:       files,
		ConfigFiles: ctx.Config.NFPM.ConfigFiles,
	}

	packager, err := nfpm.Get(format)
	if err != nil {
		return err
	}

	var path = filepath.Join(ctx.Config.Dist, name+"."+format)
	log.WithField("file", path).Info("creating")
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	defer w.Close() // nolint: errcheck
	if err := packager.Package(nfpm.WithDefaults(info), w); err != nil {
		return errors.Wrap(err, "nfpm failed")
	}
	if err := w.Close(); err != nil {
		return errors.Wrap(err, "could not close package file")
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.LinuxPackage,
		Name:   name + "." + format,
		Path:   path,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}
