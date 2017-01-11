package compress

import (
	"log"
	"os"

	"github.com/goreleaser/releaser/config"
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
func (Pipe) Run(config config.ProjectConfig) error {
	var g errgroup.Group
	for _, system := range config.Build.Oses {
		for _, arch := range config.Build.Arches {
			system := system
			arch := arch
			g.Go(func() error {
				return create(system, arch, config)
			})
		}
	}
	return g.Wait()
}

type Archive interface {
	Close() error
	Add(name, path string) error
}

func create(system, arch string, config config.ProjectConfig) error {
	name, err := config.ArchiveName(system, arch)
	if err != nil {
		return err
	}
	file, err := os.Create("dist/" + name + "." + config.Archive.Format)
	log.Println("Creating", file.Name(), "...")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	var archive = archiveFor(file, config.Archive.Format)
	defer func() { _ = archive.Close() }()
	for _, f := range config.Files {
		if err := archive.Add(f, f); err != nil {
			return err
		}
	}
	return archive.Add(config.BinaryName+extFor(system), "dist/"+name+"/"+config.BinaryName)
}

func archiveFor(file *os.File, format string) Archive {
	if format == "zip" {
		return zip.New(file)
	}
	return tar.New(file)
}

func extFor(system string) string {
	if system == "windows" {
		return ".exe"
	}
	return ""
}
