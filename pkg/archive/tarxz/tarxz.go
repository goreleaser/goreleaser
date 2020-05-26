// Package tarxz implements the Archive interface providing tar.xz archiving
// and compression.
package tarxz

import (
	"archive/tar"
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

// Archive as tar.xz.
type Archive struct {
	xzw *xz.Writer
	tw  *tar.Writer
}

// Close all closeables.
func (a Archive) Close() error {
	if err := a.tw.Close(); err != nil {
		return err
	}
	return a.xzw.Close()
}

// New tar.xz archive.
func New(target io.Writer) Archive {
	xzw, _ := xz.WriterConfig{DictCap: 16 * 1024 * 1024}.NewWriter(target)
	tw := tar.NewWriter(xzw)
	return Archive{
		xzw: xzw,
		tw:  tw,
	}
}

// Add file to the archive.
func (a Archive) Add(name, path string) error {
	file, err := os.Open(path) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, name)
	if err != nil {
		return err
	}
	header.Name = name
	if err = a.tw.WriteHeader(header); err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	_, err = io.Copy(a.tw, file)
	return err
}
