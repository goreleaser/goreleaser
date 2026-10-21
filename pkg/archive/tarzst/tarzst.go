// Package tarzst implements the Archive interface providing tar.zst archiving
// and compression.
package tarzst

import (
	"fmt"
	"io"

	"github.com/goreleaser/goreleaser/v2/pkg/archive/tar"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/klauspost/compress/zstd"
)

// Archive as tar.zst.
type Archive struct {
	zstw *zstd.Encoder
	tw   *tar.Archive
}

// New creates a tar.zst archive with the given configuration.
func New(target io.Writer, cfg config.Zstd) (Archive, error) {
	opts, err := encoderOptions(cfg)
	if err != nil {
		return Archive{}, err
	}
	zstw, err := zstd.NewWriter(target, opts...)
	if err != nil {
		return Archive{}, fmt.Errorf("create zstd encoder: %w", err)
	}
	tw := tar.New(zstw)
	return Archive{
		zstw: zstw,
		tw:   &tw,
	}, nil
}

func encoderOptions(cfg config.Zstd) ([]zstd.EOption, error) {
	var opts []zstd.EOption
	if cfg.Level != "" {
		var level zstd.EncoderLevel
		switch cfg.Level {
		case "fastest":
			level = zstd.SpeedFastest
		case "default":
			level = zstd.SpeedDefault
		case "better":
			level = zstd.SpeedBetterCompression
		case "best":
			level = zstd.SpeedBestCompression
		default:
			return nil, fmt.Errorf("invalid zstd level %q: must be fastest, default, better, or best", cfg.Level)
		}
		opts = append(opts, zstd.WithEncoderLevel(level))
	}
	if cfg.WindowSize != nil {
		if *cfg.WindowSize < 10 || *cfg.WindowSize > 29 {
			return nil, fmt.Errorf("invalid zstd window_size %d: must be between 10 and 29 (log2 bytes)", *cfg.WindowSize)
		}
		opts = append(opts, zstd.WithWindowSize(1<<*cfg.WindowSize))
	}
	return opts, nil
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
