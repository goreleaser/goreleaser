// Package archive provides tar.gz and zip archiving
package archive

import (
	"fmt"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/pkg/archive/gzip"
	"github.com/goreleaser/goreleaser/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/pkg/archive/targz"
	"github.com/goreleaser/goreleaser/pkg/archive/tarxz"
	"github.com/goreleaser/goreleaser/pkg/archive/tarzst"
	"github.com/goreleaser/goreleaser/pkg/archive/zip"
	"github.com/goreleaser/goreleaser/pkg/config"
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
	case "tar.zst":
		return tarzst.New(w), nil
	case "zip":
		return zip.New(w), nil
	}
	return nil, fmt.Errorf("invalid archive format: %s", format)
}

// Copying copies the source archive into a new one, which can be appended at.
// Source needs to be in the specified format.
func Copying(r *os.File, w io.Writer, format string) (Archive, error) {
	switch format {
	case "tar.gz", "tgz":
		return targz.Copying(r, w)
	case "tar":
		return tar.Copying(r, w)
	case "zip":
		return zip.Copying(r, w)
	}
	return nil, fmt.Errorf("invalid archive format: %s", format)
}
