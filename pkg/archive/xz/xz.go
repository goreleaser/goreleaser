// Package xz implements the Archive interface providing xz archiving
// and compression.
package xz

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/ulikunitz/xz"
)

// Archive as xz.
type Archive struct {
	name    string
	modTime time.Time
	xzw     *xz.Writer
}

// New xz archive.
func New(target io.Writer) *Archive {
	xzw, _ := xz.WriterConfig{DictCap: 16 * 1024 * 1024}.NewWriter(target)
	return &Archive{
		xzw: xzw,
	}
}

// Close all closeables.
func (a *Archive) Close() error {
	return a.xzw.Close()
}

// Add file to the archive.
func (a *Archive) Add(f config.File) error {
	if a.name != "" {
		return fmt.Errorf("xz: failed to add %s, only one file can be archived in xz format", f.Destination)
	}

	file, err := os.Open(f.Source) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	a.name = f.Destination
	_, err = io.Copy(a.xzw, file)
	return err
}
