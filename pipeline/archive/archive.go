// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/ext"
	"github.com/goreleaser/goreleaser/internal/tar"
	"github.com/goreleaser/goreleaser/internal/zip"
	"github.com/mattn/go-zglob"
	"golang.org/x/sync/errgroup"
)

// Pipe for archive
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Creating archives"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	var g errgroup.Group
	for platform, archive := range ctx.Archives {
		archive := archive
		platform := platform
		g.Go(func() error {
			if ctx.Config.Archive.Skip {
				return skip(ctx, platform, archive)
			}
			return create(ctx, platform, archive)
		})
	}
	return g.Wait()
}

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(name, path string) error
}

func create(ctx *context.Context, platform, name string) error {
	var folder = filepath.Join(ctx.Config.Dist, name)
	var format = formatFor(ctx, platform)
	file, err := os.Create(folder + "." + format)
	if err != nil {
		return err
	}
	log.Println("Creating", file.Name())
	defer func() { _ = file.Close() }()
	var archive = archiveFor(file, format)
	defer func() { _ = archive.Close() }()

	files, err := findFiles(ctx)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err = archive.Add(f, f); err != nil {
			return err
		}
	}
	var binary = ctx.Config.Build.Binary + ext.For(platform)
	if err := archive.Add(binary, filepath.Join(folder, binary)); err != nil {
		return err
	}
	ctx.AddArtifact(file.Name())
	return nil
}

func skip(ctx *context.Context, platform, name string) error {
	log.Println("Skip archiving")
	var binary = filepath.Join(ctx.Config.Dist, name+ext.For(platform))
	ctx.AddArtifact(binary)
	return nil
}

func findFiles(ctx *context.Context) (result []string, err error) {
	for _, glob := range ctx.Config.Archive.Files {
		files, err := zglob.Glob(glob)
		if err != nil {
			return result, err
		}
		result = append(result, files...)
	}
	return
}

func archiveFor(file *os.File, format string) Archive {
	if format == "zip" {
		return zip.New(file)
	}
	return tar.New(file)
}

func formatFor(ctx *context.Context, platform string) string {
	for _, override := range ctx.Config.Archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return ctx.Config.Archive.Format
}
