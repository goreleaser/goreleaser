package compress

import (
	"log"
	"os"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/pipeline/compress/tar"
	"github.com/goreleaser/releaser/pipeline/compress/zip"
	"github.com/goreleaser/releaser/uname"
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
	file, err := os.Create("dist/" + nameFor(system, arch, config.BinaryName) + "." + config.Archive.Format)
	log.Println("Creating", file.Name(), "...")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	var archive Archive
	if config.Archive.Format == "zip" {
		archive = zip.New(file)
	} else {
		archive = tar.New(file)
	}
	defer func() { _ = archive.Close() }()
	for _, f := range config.Files {
		if err := archive.Add(f, f); err != nil {
			return err
		}
	}
	return archive.Add(config.BinaryName+ext(system), binaryPath(system, arch, config.BinaryName))
}

func nameFor(system, arch, binary string) string {
	return binary + "_" + uname.FromGo(system) + "_" + uname.FromGo(arch)
}

func binaryPath(system, arch, binary string) string {
	return "dist/" + nameFor(system, arch, binary) + "/" + binary
}

func ext(system string) string {
	if system == "windows" {
		return ".exe"
	}
	return ""
}
