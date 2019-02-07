// Package archive provides tar.gz and zip archiving
package archive

import (
	"os"

	"path/filepath"

	"github.com/goreleaser/goreleaser/pkg/archive/gzip"
	"github.com/goreleaser/goreleaser/pkg/archive/targz"
	"github.com/goreleaser/goreleaser/pkg/archive/zip"
)

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(name, path string) error
}

// New archive
// Defaults to tar.gz. Zip and GZip formats are detected from the file extension.
func New(file *os.File) Archive {
	switch filepath.Ext(file.Name()) {
	case ".zip":
		return zip.New(file)
	case ".gz":
		return gzip.New(file)
	default:
		return targz.New(file)
	}
}
