// Package zip implements the Archive interface providing zip archiving
// and compression.
package zip

import (
	"archive/zip"
	"compress/flate"
	"io"
	"os"

	"github.com/goreleaser/goreleaser/pkg/config"
)

// Archive zip struct.
type Archive struct {
	z *zip.Writer
}

// Close all closeables.
func (a Archive) Close() error {
	return a.z.Close()
}

// New zip archive.
func New(target io.Writer) Archive {
	compressor := zip.NewWriter(target)
	compressor.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	return Archive{
		z: compressor,
	}
}

// Add a file to the zip archive.
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
	if info.IsDir() {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = f.Destination
	header.Method = zip.Deflate
	header.Name = f.Destination
	if !f.FileInfo.MTime.IsZero() {
		header.Modified = f.FileInfo.MTime
	}
	if f.FileInfo.Mode != 0 {
		header.SetMode(f.FileInfo.Mode)
	}
	w, err := a.z.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, file)
	return err
}

// TODO: test fileinfo stuff
