package gio

import (
	"fmt"
	"io"
	"os"
)

// CopyFileWithSrcMode copies src into dst with src's file mode.
func CopyFileWithSrcMode(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", src, err)
	}
	return CopyFile(src, dst, st.Mode())
}

// CopyFile copies src into dst with the given mode.
func CopyFile(src, dst string, mode os.FileMode) error {
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
