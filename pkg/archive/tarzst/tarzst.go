// Package tarzst implements the Archive interface providing tar.zst archiving
// and compression.
package tarzst

import (
	"io"

	"github.com/goreleaser/goreleaser/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/klauspost/compress/zstd"
)

// Archive as tar.zst.
type Archive struct {
	zstw *zstd.Encoder
	tw   *tar.Archive
}

// New tar.zst archive.
func New(target io.Writer) Archive {
	zstw, _ := zstd.NewWriter(target)
	tw := tar.New(zstw)
	return Archive{
		zstw: zstw,
		tw:   &tw,
	}
}

// Close all closeables.
func (a Archive) Close() error {
	if err := a.tw.Close(); err != nil {
		return err
	}
	return a.zstw.Close()
}

// Add file to the archive.
func (a Archive) Add(f config.File) error {
	return a.tw.Add(f)
}
