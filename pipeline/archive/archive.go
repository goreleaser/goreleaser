// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/mattn/go-zglob"
	"golang.org/x/sync/errgroup"

	"github.com/goreleaser/archive"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/filenametemplate"
)

const (
	defaultNameTemplate       = "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	defaultBinaryNameTemplate = "{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
)

// Pipe for archive
type Pipe struct{}

func (Pipe) String() string {
	return "creating archives"
}

// Default sets the pipe defaults
func (Pipe) Default(ctx *context.Context) error {
	var archive = &ctx.Config.Archive
	if archive.Format == "" {
		archive.Format = "tar.gz"
	}
	if len(archive.Files) == 0 {
		archive.Files = []string{
			"licence*",
			"LICENCE*",
			"license*",
			"LICENSE*",
			"readme*",
			"README*",
			"changelog*",
			"CHANGELOG*",
		}
	}
	if archive.NameTemplate == "" {
		if archive.Format == "binary" {
			archive.NameTemplate = defaultBinaryNameTemplate
		} else {
			archive.NameTemplate = defaultNameTemplate
		}
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	var filtered = ctx.Artifacts.Filter(artifact.ByType(artifact.Binary))
	for _, artifacts := range filtered.GroupByPlatform() {
		artifacts := artifacts
		g.Go(func() error {
			if ctx.Config.Archive.Format == "binary" {
				return skip(ctx, artifacts)
			}
			return create(ctx, artifacts)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, binaries []artifact.Artifact) error {
	var format = packageFormat(ctx, binaries[0].Goos)
	folder, err := filenametemplate.Apply(
		ctx.Config.Archive.NameTemplate,
		filenametemplate.NewFields(ctx, ctx.Config.Archive.Replacements, binaries...),
	)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %s", archivePath, err.Error())
	}
	defer archiveFile.Close() // nolint: errcheck
	log.WithField("archive", archivePath).Info("creating")
	var a = archive.New(archiveFile)
	defer a.Close() // nolint: errcheck

	files, err := findFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to find files to archive: %s", err.Error())
	}
	for _, f := range files {
		if err = a.Add(wrap(ctx, f, folder), f); err != nil {
			return fmt.Errorf("failed to add %s to the archive: %s", f, err.Error())
		}
	}
	for _, binary := range binaries {
		if err := a.Add(wrap(ctx, binary.Name, folder), binary.Path); err != nil {
			return fmt.Errorf("failed to add %s -> %s to the archive: %s", binary.Path, binary.Name, err.Error())
		}
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   folder + "." + format,
		Path:   archivePath,
		Goos:   binaries[0].Goos,
		Goarch: binaries[0].Goarch,
		Goarm:  binaries[0].Goarm,
	})
	return nil
}

func skip(ctx *context.Context, binaries []artifact.Artifact) error {
	for _, binary := range binaries {
		log.WithField("binary", binary.Name).Info("skip archiving")
		var fields = filenametemplate.NewFields(ctx, ctx.Config.Archive.Replacements, binary)
		name, err := filenametemplate.Apply(ctx.Config.Archive.NameTemplate, fields)
		if err != nil {
			return err
		}
		binary.Type = artifact.UploadableBinary
		binary.Name = name + binary.Extra["Ext"]
		ctx.Artifacts.Add(binary)
	}
	return nil
}

func findFiles(ctx *context.Context) (result []string, err error) {
	for _, glob := range ctx.Config.Archive.Files {
		files, err := zglob.Glob(glob)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %s", glob, err.Error())
		}
		result = append(result, files...)
	}
	return
}

// Wrap archive files with folder if set in config.
func wrap(ctx *context.Context, name, folder string) string {
	if ctx.Config.Archive.WrapInDirectory {
		return filepath.Join(folder, name)
	}
	return name
}

func packageFormat(ctx *context.Context, platform string) string {
	for _, override := range ctx.Config.Archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return ctx.Config.Archive.Format
}
