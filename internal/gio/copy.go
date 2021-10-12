package gio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
)

// Copy recursively copies src into dst with src's file modes.
func Copy(src, dst string) error {
	return CopyWithMode(src, dst, 0)
}

// CopyWithMode recursively copies src into dst with the given mode.
// The given mode applies only to files. Their parent dirs will have the same mode as their src counterparts.
func CopyWithMode(src, dst string, mode os.FileMode) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
		}
		// We have the following:
		// - src = "a/b"
		// - dst = "dist/linuxamd64/b"
		// - path = "a/b/c.txt"
		// So we join "a/b" with "c.txt" and use it as the destination.
		dst := filepath.Join(dst, strings.Replace(path, src, "", 1))
		log.WithFields(log.Fields{
			"src": path,
			"dst": dst,
		}).Debug("copying file")
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return copySymlink(path, dst)
		}
		if mode != 0 {
			return copyFile(path, dst, mode)
		}
		return copyFile(path, dst, info.Mode())
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

	new, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", dst, err)
	}
	defer new.Close()

	if _, err := io.Copy(new, original); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}
	return nil
}
