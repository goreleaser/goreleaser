// Package tarxz implements the Archive interface providing tar.xz archiving
// and compression.
package tarxz

import (
	"archive/tar"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/pkg/config"
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
func (a Archive) Add(f config.File) error {
	file, err := os.Open(f.Source) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, f.Destination)
	if err != nil {
		return err
	}
	header.Name = f.Destination
	if !f.FileInfo.MTime.IsZero() {
		header.ModTime = f.FileInfo.MTime
	}
	if f.FileInfo.Mode != 0 {
		header.Mode = int64(f.FileInfo.Mode)
	}
	if f.FileInfo.Owner != "" {
		header.Uid = 0
		header.Uname = f.FileInfo.Owner
	}
	if f.FileInfo.Group != "" {
		header.Gid = 0
		header.Gname = f.FileInfo.Group
	}
	if err = a.tw.WriteHeader(header); err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	_, err = io.Copy(a.tw, file)
	return err
}
