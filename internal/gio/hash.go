package gio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// EqualFiles returns true if both files sha256sums and their modes are equal.
func EqualFiles(a, b string) (bool, error) {
	am, as, err := sha256sum(a)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %w", a, err)
	}
	bm, bs, err := sha256sum(b)
	if err != nil {
		return false, fmt.Errorf("could not hash %s: %w", b, err)
	}
	return as == bs && am == bm, nil
}

func sha256sum(path string) (fs.FileMode, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return 0, "", err
	}

	st, err := f.Stat()
	if err != nil {
		return 0, "", err
	}

	return st.Mode(), hex.EncodeToString(h.Sum(nil)), nil
}
