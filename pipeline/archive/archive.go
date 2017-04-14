// Package archive implements the pipe interface with the intent of
// archiving and compressing the binaries, readme, and other artifacts. It
// also provides an Archive interface which represents an archiving format.
package archive

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline/archive/tar"
	"github.com/goreleaser/goreleaser/pipeline/archive/zip"
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
	for _, archive := range ctx.Archives {
		archive := archive
		g.Go(func() error {
			return create(ctx, archive)
		})
	}
	return g.Wait()
}

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(name, path string) error
}

func create(ctx *context.Context, name string) error {
	folder := filepath.Join(ctx.Config.Dist, name)
	file, err := os.Create(folder + "." + ctx.Config.Archive.Format)
	log.Println("Creating", file.Name())
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	var archive = archiveFor(file, ctx.Config.Archive.Format)
	defer func() { _ = archive.Close() }()
	for _, f := range ctx.Config.Archive.Files {
		if err = archive.Add(f, f); err != nil {
			return err
		}
	}
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := archive.Add(f.Name(), filepath.Join(folder, f.Name())); err != nil {
			return err
		}
	}
	ctx.AddArtifact(file.Name())
	return nil
}

func archiveFor(file *os.File, format string) Archive {
	if format == "zip" {
		return zip.New(file)
	}
	return tar.New(file)
}
