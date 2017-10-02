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
		archivePath := filepath.Join(ctx.Config.Dist, folder+"."+format)
		archiveFile, err := os.Create(archivePath)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %s", archivePath, err.Error())
		}
		defer func() {
			if e := archiveFile.Close(); e != nil {
				log.WithField("archive", archivePath).Errorf("failed to close file: %v", e)
			}
		}()
		log.WithField("archive", archivePath).Info("creating")
		var a = archive.New(archiveFile)
		defer func() {
			if e := a.Close(); e != nil {
				log.WithField("archive", archivePath).Errorf("failed to close archive: %v", e)
			}
		}()

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
		ctx.AddArtifact(archivePath)
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

// Wrap archive files with folder if set in config.
func wrap(ctx *context.Context, name, folder string) string {
	if ctx.Config.Archive.WrapInDirectory {
		return filepath.Join(folder, name)
	}
	return name
}
