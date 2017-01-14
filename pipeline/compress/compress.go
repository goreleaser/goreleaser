package compress

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/goreleaser/releaser/context"
	"github.com/goreleaser/releaser/pipeline/compress/tar"
	"github.com/goreleaser/releaser/pipeline/compress/zip"
	"golang.org/x/sync/errgroup"
)

// Pipe for compress
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Compress"
}

// Run the pipe
func (Pipe) Run(context *context.Context) error {
	var g errgroup.Group
	for _, archive := range context.Archives {
		archive := archive
		g.Go(func() error {
			return create(archive, context)
		})
	}
	return g.Wait()
}

type Archive interface {
	Close() error
	Add(name, path string) error
}

func create(archive string, context *context.Context) error {
	file, err := os.Create("dist/" + archive + "." + context.Config.Archive.Format)
	log.Println("Creating", file.Name(), "...")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	var archive = archiveFor(file, context.Config.Archive.Format)
	defer func() { _ = archive.Close() }()
	for _, f := range context.Config.Files {
		if err := archive.Add(f, f); err != nil {
			return err
		}
	}
	files, err := ioutil.ReadDir("dist/" + name)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := archive.Add(file.Name(), f); err != nil {
			return err
		}
	}
	return nil
}

func archiveFor(file *os.File, format string) Archive {
	if format == "zip" {
		return zip.New(file)
	}
	return tar.New(file)
}
