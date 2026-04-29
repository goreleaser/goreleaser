package nodesea

import (
	"context"
	"fmt"
	"os"
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
	if err := copyFile(cachedPath, outPath); err != nil {
		return err
	}
	if err := injectELF(outPath, blob); err != nil {
		return fmt.Errorf("nodesea: inject elf: %w", err)
	}
	return nil
}

// buildPE reads cachedPath, strips its Authenticode signature, injects
// blob as a NODE_SEA_BLOB resource, flips the sentinel, and writes
// outPath atomically with executable permissions.
func buildPE(cachedPath, outPath string, blob []byte) error {
	if err := copyFile(cachedPath, outPath); err != nil {
		return err
	}
	if err := unsignPE(outPath); err != nil {
		return fmt.Errorf("nodesea: unsign pe: %w", err)
	}
	if err := injectPE(outPath, blob); err != nil {
		return fmt.Errorf("nodesea: inject pe: %w", err)
	}
	return nil
}

// copyFile copies src to dst with executable permissions, replacing
// dst if it already exists.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}
