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

// Close all closeables.
func (a Archive) Close() error {
	return a.z.Close()
}

// Add a file to the zip archive.
func (a Archive) Add(f config.File) error {
	info, err := os.Lstat(f.Source) // #nosec
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
	if !f.Info.MTime.IsZero() {
		header.Modified = f.Info.MTime
	}
	if f.Info.Mode != 0 {
		header.SetMode(f.Info.Mode)
	}
	w, err := a.z.CreateHeader(header)
	if err != nil {
		return err
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	file, err := os.Open(f.Source) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(w, file)
	return err
}

// TODO: test fileinfo stuff
