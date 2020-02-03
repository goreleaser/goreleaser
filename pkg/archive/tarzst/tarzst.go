// Package tarzst implements the Archive interface providing tar.zst archiving
// and compression.
package tarzst

import (
	"archive/tar"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

// Archive as tar.zst
type Archive struct {
	zw *zstd.Encoder
	tw *tar.Writer
}

// Close all closeables
func (a Archive) Close() error {
	if err := a.tw.Close(); err != nil {
		return err
	}
	return a.zw.Close()
}

// New tar.zst archive
func New(target io.Writer) Archive {
	zw, _ := zstd.NewWriter(target)
	tw := tar.NewWriter(zw)
	return Archive{
		zw: zw,
		tw: tw,
	}
}

// Add file to the archive
func (a Archive) Add(name, path string) error {
	file, err := os.Open(path) // #nosec
	if err != nil {
		return err
	}
	defer file.Close() // nolint: errcheck
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
