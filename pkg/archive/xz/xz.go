// Package xz implements the Archive interface providing xz archiving
// and compression.
package xz

import (
	//"fmt"
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

// Archive as xz
type Archive struct {
	xzw *xz.Writer
}

// Close all closeables
func (a Archive) Close() error {
	return a.xzw.Close()
}

// New xz archive
func New(target io.Writer) Archive {
	xzw, _ := xz.NewWriter(target)
	return Archive{
		xzw: xzw,
	}
}

// Add file to the archive
func (a Archive) Add(name, path string) error {
	/*
	if a.xzw.Header.Name != "" {
		return fmt.Errorf("xz: failed to add %s, only one file can be archived in xz format", name)
	} */
	file, err := os.Open(path) // #nosec
	if err != nil {
		return err
	}
	defer file.Close() // nolint: errcheck
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	//a.xzw.Header.Name = name
	//a.xzw.Header.ModTime = info.ModTime()
	_, err = io.Copy(a.xzw, file)
	return err
}
