// Package archive provides tar.gz and zip archiving
package archive

import (
	"fmt"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/v2/pkg/archive/gzip"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/makeself"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/targz"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/tarxz"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/tarzst"
	"github.com/goreleaser/goreleaser/v2/pkg/archive/zip"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(f config.File) error
}

// New archive.
func New(w io.Writer, format string) (Archive, error) {
	switch format {
	case "tar.gz", "tgz":
		return targz.New(w), nil
	case "tar":
		return tar.New(w), nil
	case "gz":
		return gzip.New(w), nil
	case "tar.xz", "txz":
		return tarxz.New(w), nil
	case "tar.zst", "tzst":
		return tarzst.New(w), nil
	case "zip":
		return zip.New(w), nil
	case "makeself":
		return makeself.New(w), nil
	}
	return nil, fmt.Errorf("invalid archive format: %s", format)
}

// NewWithMakeselfConfig creates a makeself archive with custom configuration.
func NewWithMakeselfConfig(w io.Writer, outputPath string, cfg config.MakeselfConfig) (Archive, error) {
	makeselfCfg := makeself.MakeselfConfig{
		OutputPath:        outputPath,
		InstallScript:     cfg.InstallScript,
		InstallScriptFile: cfg.InstallScriptFile,
		Label:             cfg.Label,
		Compression:       cfg.Compression,
		ExtraArgs:         cfg.ExtraArgs,
		LSMTemplate:       cfg.LSMTemplate,
		LSMFile:           cfg.LSMFile,
	}
	return makeself.NewWithConfig(w, outputPath, makeselfCfg), nil
}

// Copy copies the source archive into a new one, which can be appended at.
// Source needs to be in the specified format.
func Copy(r *os.File, w io.Writer, format string) (Archive, error) {
	switch format {
	case "tar.gz", "tgz":
		return targz.Copy(r, w)
	case "tar":
		return tar.Copy(r, w)
	case "zip":
		return zip.Copy(r, w)
	}
	return nil, fmt.Errorf("invalid archive format: %s", format)
}
