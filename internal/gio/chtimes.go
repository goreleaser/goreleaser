package gio

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Chtimes applies the given ts to the given path.
func Chtimes(path, ts string) error {
	if ts == "" {
		return nil
	}
	modUnix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("chtimes: %s: %w", path, err)
	}
	modTime := time.Unix(modUnix, 0)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		return fmt.Errorf("chtimes: %s: %w", path, err)
	}
	return nil
}
