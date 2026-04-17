package nodesea

import (
	"context"
	"fmt"
	"os"
)

// Inject dispatches to the format-appropriate injector for the given
// target binary. It is a no-op error when target is not one of the three
// supported formats.
func Inject(target Target, hostPath string, blob []byte) error {
	switch FormatFor(target.Goos()) {
	case FormatELF:
		return InjectELF(hostPath, blob)
	case FormatMachO:
		return InjectMachO(hostPath, blob)
	case FormatPE:
		return InjectPE(hostPath, blob)
	default:
		return fmt.Errorf("%w: target %q has no SEA injector", ErrNotSupported, target)
	}
}

// Unsign strips the existing signature, if any, from the host binary
// at hostPath. No-op for ELF.
func Unsign(target Target, hostPath string) error {
	switch FormatFor(target.Goos()) {
	case FormatMachO:
		return UnsignMachO(hostPath)
	case FormatPE:
		return UnsignPE(hostPath)
	default:
		return nil
	}
}

// PrepareHost ensures a host binary for (version, target) is available
// and ready to be injected into. It downloads and caches the binary
// (verifying its SHA256), then copies it into outPath and strips the
// existing signature.
//
// If outPath already exists it is overwritten. Returns the same path on
// success.
func PrepareHost(ctx context.Context, version string, target Target, outPath string) (string, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", err
	}
	cachedPath, err := downloadHost(ctx, cacheDir, version, target)
	if err != nil {
		return "", err
	}

	src, err := os.ReadFile(cachedPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, src, 0o755); err != nil {
		return "", err
	}
	if err := Unsign(target, outPath); err != nil {
		return "", fmt.Errorf("nodesea: unsign host: %w", err)
	}
	return outPath, nil
}
