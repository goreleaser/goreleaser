// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goreleaser/archive"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/ext"
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
			return create(ctx, platform, archive)
		})
	}
	return g.Wait()
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

func archiveFor(file *os.File, format string) archive.Archive {
	if format == "zip" {
		return archive.NewZip(file)
	}
	return archive.NewTargz(file)
}

func formatFor(ctx *context.Context, platform string) string {
	for _, override := range ctx.Config.Archive.FormatOverrides {
		if strings.HasPrefix(platform, override.Goos) {
			return override.Format
		}
	}
	return ctx.Config.Archive.Format
}
