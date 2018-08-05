// Package archive provides tar.gz and zip archiving
package archive

import (
	"os"

	"path/filepath"

	"github.com/goreleaser/goreleaser/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/pkg/archive/zip"
)

// Archive represents a compression archive files from disk can be written to.
type Archive interface {
	Close() error
	Add(name, path string) error
}

// New archive
// If the exentions of the target file is .zip, the archive will be in the zip
// format, otherwise, it will be a tar.gz archive.
func New(file *os.File) Archive {
	if filepath.Ext(file.Name()) == ".zip" {
		return zip.New(file)
	}
	return tar.New(file)
}
