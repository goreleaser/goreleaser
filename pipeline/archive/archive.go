// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/archive"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/archiveformat"
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
	for platform, binaries := range ctx.Binaries {
		platform := platform
		binaries := binaries
		g.Go(func() error {
			if ctx.Config.Archive.Format == "binary" {
				return skip(ctx, platform, binaries)
			}
			return create(ctx, platform, binaries)
		})
	}
	return g.Wait()
}

func create(ctx *context.Context, platform string, groups map[string][]context.Binary) error {
	for folder, binaries := range groups {
		var format = archiveformat.For(ctx, platform)
		targetFolder := filepath.Join(ctx.Config.Dist, folder+"."+format)
		file, err := os.Create(targetFolder)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %s", targetFolder, err.Error())
		}
		defer func() { _ = file.Close() }()
		log.WithField("archive", file.Name()).Info("creating")
		var archive = archive.New(file)
		defer func() { _ = archive.Close() }()

		files, err := findFiles(ctx)
		if err != nil {
			return fmt.Errorf("failed to find files to archive: %s", err.Error())
		}
		for _, f := range files {
			if err = archive.Add(f, f); err != nil {
				return fmt.Errorf("failed to add %s to the archive: %s", f, err.Error())
			}
		}
		for _, binary := range binaries {
			if err := archive.Add(binary.Name, binary.Path); err != nil {
				return fmt.Errorf("failed to add %s -> %s to the archive: %s", binary.Path, binary.Name, err.Error())
			}
		}
		ctx.AddArtifact(file.Name())
	}
	return nil
}

func skip(ctx *context.Context, platform string, groups map[string][]context.Binary) error {
	for _, binaries := range groups {
		for _, binary := range binaries {
			log.WithField("binary", binary.Name).Info("skip archiving")
			ctx.AddArtifact(binary.Path)
		}
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
