// Package archive provides tar.gz and zip archiving
package archive

import (
	"fmt"
	"io"

	"github.com/goreleaser/goreleaser/pkg/archive/gzip"
	"github.com/goreleaser/goreleaser/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/pkg/archive/targz"
	"github.com/goreleaser/goreleaser/pkg/archive/tarxz"
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
	case "tar.gz":
		return targz.New(w), nil
	case "tar":
		return tar.New(w), nil
	case "gz":
		return gzip.New(w), nil
	case "tar.xz":
		return tarxz.New(w), nil
	case "zip":
		return zip.New(w), nil
	}
	return nil, fmt.Errorf("invalid archive format: %s", format)
}
