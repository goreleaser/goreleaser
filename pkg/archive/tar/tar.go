// Package tar implements the Archive interface providing tar archiving.
package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/pkg/config"
)

// Archive as tar.
type Archive struct {
	tw *tar.Writer
}

// New tar archive.
func New(target io.Writer) Archive {
	return Archive{
		tw: tar.NewWriter(target),
	}
}

// Close all closeables.
func (a Archive) Close() error {
	return a.tw.Close()
}

// Add file to the archive.
func (a Archive) Add(f config.File) error {
	info, err := os.Lstat(f.Source) // #nosec
	if err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	var link string
	if info.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(f.Source) // #nosec
		if err != nil {
			return fmt.Errorf("%s: %w", f.Source, err)
		}
	}
	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	header.Name = f.Destination
	if !f.Info.MTime.IsZero() {
		header.ModTime = f.Info.MTime
	}
	if f.Info.Mode != 0 {
		header.Mode = int64(f.Info.Mode)
	}
	if f.Info.Owner != "" {
		header.Uid = 0
		header.Uname = f.Info.Owner
	}
	if f.Info.Group != "" {
		header.Gid = 0
		header.Gname = f.Info.Group
	}
	if err = a.tw.WriteHeader(header); err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	file, err := os.Open(f.Source) // #nosec
	if err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	defer file.Close()
	if _, err := io.Copy(a.tw, file); err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	return nil
}
