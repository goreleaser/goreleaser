package compress

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"

	"github.com/goreleaser/releaser/config"
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

func create(system, arch string, config config.ProjectConfig) error {
	file, err := os.Create("dist/" + nameFor(system, arch, config.BinaryName) + ".tar.gz")
	log.Println("Creating", file.Name(), "...")
	if err != nil {
		return err
	}
	defer file.Close()
	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, f := range config.Files {
		if err := addFile(tw, f, f); err != nil {
			return err
		}
	}
	return addFile(tw, config.BinaryName+ext(system), binaryPath(system, arch, config.BinaryName))
}

func addFile(tw *tar.Writer, name, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	header := new(tar.Header)
	header.Name = name
	header.Size = stat.Size()
	header.Mode = int64(stat.Mode())
	header.ModTime = stat.ModTime()
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, file); err != nil {
		return err
	}
	return nil
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
