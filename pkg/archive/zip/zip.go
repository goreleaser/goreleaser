// Package zip implements the Archive interface providing zip archiving
// and compression.
package zip

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/pkg/config"
)

// Archive zip struct.
type Archive struct {
	z     *zip.Writer
	files map[string]bool
}

// New zip archive.
func New(target io.Writer) Archive {
	compressor := zip.NewWriter(target)
	compressor.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	return Archive{
		z:     compressor,
		files: map[string]bool{},
	}
}

// New zip archive.
func Copying(source *os.File, target io.Writer) (Archive, error) {
	info, err := source.Stat()
	if err != nil {
		return Archive{}, err
	}
	r, err := zip.NewReader(source, info.Size())
	if err != nil {
		return Archive{}, err
	}
	w := New(target)
	for _, zf := range r.File {

		w.files[zf.Name] = true
		hdr := zip.FileHeader{
			Name:               zf.Name,
			UncompressedSize64: zf.UncompressedSize64,
			UncompressedSize:   zf.UncompressedSize,
			CreatorVersion:     zf.CreatorVersion,
			ExternalAttrs:      zf.ExternalAttrs,
		}
		ww, err := w.z.CreateHeader(&hdr)
		if err != nil {
			return Archive{}, fmt.Errorf("creating %q header in target: %w", zf.Name, err)
		}
		if zf.Mode().IsDir() {
			continue
		}
		rr, err := zf.Open()
		if err != nil {
			return Archive{}, fmt.Errorf("opening %q from source: %w", zf.Name, err)
		}
		defer rr.Close()
		if _, err = io.Copy(ww, rr); err != nil {
			return Archive{}, fmt.Errorf("copy from %q source to target: %w", zf.Name, err)
		}
		_ = rr.Close()
	}
	return w, nil
}

// Close all closeables.
func (a Archive) Close() error {
	return a.z.Close()
}

// Add a file to the zip archive.
func (a Archive) Add(f config.File) error {
	if _, ok := a.files[f.Destination]; ok {
		return &fs.PathError{Err: fs.ErrExist, Path: f.Destination, Op: "add"}
	}
	a.files[f.Destination] = true
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
	if !f.Info.ParsedMTime.IsZero() {
		header.Modified = f.Info.ParsedMTime
	}
	if f.Info.Mode != 0 {
		header.SetMode(f.Info.Mode)
	}
	w, err := a.z.CreateHeader(header)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(f.Source) // #nosec
		if err != nil {
			return fmt.Errorf("%s: %w", f.Source, err)
		}
		_, err = io.WriteString(w, filepath.ToSlash(link))
		return err
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
