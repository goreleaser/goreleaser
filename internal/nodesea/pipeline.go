package nodesea

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Build produces a Node.js Single Executable Application at outPath for
// (version, target) by downloading the official Node host binary into
// the user cache, then splicing blob into a private SEA segment/section
// in a single in-memory pass per format.
//
// One call replaces what was previously download + copy + unsign +
// inject + sentinel-flip + (Mach-O only) ad-hoc sign. outPath is
// written atomically: on success it has executable permissions
// regardless of any pre-existing file at that path.
func Build(ctx context.Context, version string, target Target, outPath string, blob []byte) error {
	cacheDir, err := CacheDir()
	if err != nil {
		return err
	}
	cachedPath, err := downloadHost(ctx, cacheDir, version, target)
	if err != nil {
		return err
	}

	switch FormatFor(target.Goos()) {
	case FormatELF:
		return buildELF(cachedPath, outPath, blob)
	case FormatMachO:
		return buildMachO(cachedPath, outPath, blob, "")
	case FormatPE:
		return buildPE(cachedPath, outPath, blob)
	default:
		return fmt.Errorf("%w: target %q has no SEA injector", ErrNotSupported, target)
	}
}

// buildELF reads cachedPath, injects blob, flips the sentinel, and
// writes outPath atomically with executable permissions.
func buildELF(cachedPath, outPath string, blob []byte) error {
	data, err := os.ReadFile(cachedPath)
	if err != nil {
		return err
	}
	out, err := injectELFBytes(data, blob)
	if err != nil {
		return fmt.Errorf("nodesea: inject elf: %w", err)
	}
	return writeFileAtomic(outPath, out, 0o755)
}

// buildPE reads cachedPath, strips its Authenticode signature, injects
// blob as a NODE_SEA_BLOB resource, flips the sentinel, and writes
// outPath atomically with executable permissions.
func buildPE(cachedPath, outPath string, blob []byte) error {
	data, err := os.ReadFile(cachedPath)
	if err != nil {
		return err
	}
	out, err := unsignPEBytes(data)
	if err != nil {
		return fmt.Errorf("nodesea: unsign pe: %w", err)
	}
	out, err = injectPEBytes(out, blob)
	if err != nil {
		return fmt.Errorf("nodesea: inject pe: %w", err)
	}
	return writeFileAtomic(outPath, out, 0o755)
}

// writeFileAtomic writes data to path via a temp file in the same
// directory, then renames it into place. perm is applied to the temp
// file before the rename so the on-disk file ends up with the
// requested mode regardless of umask. On error the temp file is
// removed and path is left untouched.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
