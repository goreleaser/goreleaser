package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

// SourceNameTemplate is the default value for the source name template
const SourceNameTemplate = "{{.Binary}}-{{.Version}}"

// Pipe for archive
type Pipe struct{}

// Description of the pipe
func (Pipe) String() string {
	return "Creating source archives"
}

// Default sets the defaults
func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Source.NameTemplate == "" {
		ctx.Config.Source.NameTemplate = SourceNameTemplate
	}
	return nil
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	return create(ctx, ".")
}

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(name, path string) error
}

func create(ctx *context.Context, wd string) error {
	if len(ctx.Config.Builds) < 1 {
		return fmt.Errorf("need at least one build")
	}

	if err := os.MkdirAll(ctx.Config.Dist, 0700); err != nil {
		return err
	}
	name, err := nameFor(ctx)
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("filename must not be empty. check your name_template")
	}

	filename := name + ".tar.gz"
	path := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("filename", filename).Info("Building archive")
	if err := createTarGz(ctx, path, wd); err != nil {
		return err
	}
	ctx.Artifacts.Add(artifact.Artifact{
		Type: artifact.Source,
		Name: filename,
		Path: path,
	})
	ctx.Config.Brew.SourceTarball = filename
	return nil
}

func createTarGz(ctx *context.Context, path, wd string) error {
	// open archive file for writing
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fh.Close() // nolint: errcheck
	// set up gzip writer
	gzw := gzip.NewWriter(fh)
	defer gzw.Close() // nolint: errcheck
	// set up tar writer
	tw := tar.NewWriter(gzw)
	defer tw.Close() // nolint: errcheck

	prefix := fmt.Sprintf("%s-%s/", ctx.Config.Builds[0].Binary, ctx.Version)

	// add COMMIT file identifying the current git commit
	if err := addBytes(tw, prefix, "COMMIT", []byte(ctx.Git.Commit)); err != nil {
		return err
	}

	// add all files
	if err := addFiles(ctx, tw, path, prefix, wd); err != nil {
		return err
	}
	return nil
}

func addFiles(ctx *context.Context, tw *tar.Writer, archive, prefix, basepath string) error {
	return filepath.Walk(basepath, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && filename != basepath {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filename == basepath {
			return nil
		}
		// avoid recursive archiving
		if filename == archive {
			return nil
		}
		for _, e := range ctx.Config.Source.Excludes {
			if m, _ := filepath.Match(e, filename); m {
				return nil
			}
		}
		filename = strings.TrimPrefix(filename, basepath+string(filepath.Separator))
		return addFile(tw, prefix, filename, basepath)
	})
}

func addFile(tw *tar.Writer, prefix, name, basepath string) error {
	fh, err := os.Open(filepath.Join(basepath, name))
	if err != nil {
		return err
	}
	defer fh.Close() // nolint: errcheck
	fi, err := fh.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{}
	header.Name = filepath.Join(prefix, name)
	header.Size = fi.Size()
	header.Mode = int64(fi.Mode())
	header.ModTime = fi.ModTime()
	if err := tw.WriteHeader(header); err != nil { // nolint: vetshadow
		return err
	}
	_, err = io.Copy(tw, fh)
	return err
}

func addBytes(tw *tar.Writer, prefix, name string, body []byte) error {
	header := &tar.Header{}
	header.Name = filepath.Join(prefix, name)
	header.Size = int64(len(body))
	header.Mode = 0644
	header.ModTime = time.Now()
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := io.Copy(tw, bytes.NewReader(body))
	return err
}
