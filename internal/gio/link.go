package gio

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
)

// Link creates a hard link and the parent directory if it does not exist yet.
func Link(src, dst string) error {
	log.WithField("src", src).WithField("dst", dst).Debug("creating link")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to make dir for destination: %w", err)
	}
	return os.Link(src, dst)
}
