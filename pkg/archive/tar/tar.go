// Package tar implements the Archive interface providing tar archiving.
package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

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

type dummyFs struct {
	file fs.FileInfo
}

var _ fs.FileInfo = dummyFs{}

func (d dummyFs) IsDir() bool {
	return d.file.IsDir()
}

func (d dummyFs) ModTime() time.Time {
	return time.Time{}
}

func (d dummyFs) Mode() fs.FileMode {
	return d.file.Mode()
}

func (d dummyFs) Name() string {
	return d.file.Name()
}

func (d dummyFs) Size() int64 {
	return d.file.Size()
}

func (d dummyFs) Sys() any {
	return nil
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
	header, err := tar.FileInfoHeader(dummyFs{file: info}, link)
	if err != nil {
		return fmt.Errorf("%s: %w", f.Source, err)
	}
	header.Name = f.Destination
	// if !f.Info.MTime.IsZero() {
	// 	header.ModTime = f.Info.MTime
	// }
	// if f.Info.Mode != 0 {
	// 	header.Mode = int64(f.Info.Mode)
	// }
	// if f.Info.Owner != "" {
	// 	header.Uid = 0
	// 	header.Uname = f.Info.Owner
	// }
	// if f.Info.Group != "" {
	// 	header.Gid = 0
	// 	header.Gname = f.Info.Group
	// }
	// header.Format = tar.FormatUnknown
	// fmt.Printf("%#v\n", header)
	fmt.Fprintf(os.Stderr, "%v\n", header.Name)
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
