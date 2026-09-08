package gio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/caarlos0/log"
)

// Copy recursively copies src into dst with src's file modes.
func Copy(src, dst string) error {
	return CopyWithMode(src, dst, 0)
}

// CopyWithMode recursively copies src into dst with the given mode.
// The given mode applies only to files. Their parent dirs will have the same mode as their src counterparts.
func CopyWithMode(src, dst string, mode os.FileMode) error {
	dst = filepath.Clean(dst)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		log.WithField("src", filepath.ToSlash(path)).WithField("dst", filepath.ToSlash(target)).Debug("copying file")
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return copySymlink(path, target)
		}
		if mode != 0 {
			return copyFile(path, target, mode)
		}
		return copyFile(path, target, info.Mode())
	})
}

func copySymlink(src, dst string) error {
	src, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(src, dst)
}

func copyFile(src, dst string, mode os.FileMode) error {
	original, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", src, err)
	}
	defer original.Close()

	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", dst, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, original); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	return nil
}
