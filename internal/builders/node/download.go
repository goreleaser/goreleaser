package node

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
)

// downloadHostBinary fetches the per-target Node.js host binary for
// (version, target) and returns its absolute path. tar.gz archives
// are extracted; bare windows .exe archives are used as-is. The
// returned path lives under a fresh temp directory.
func downloadHostBinary(ctx context.Context, version string, target Target) (string, error) {
	archive, err := nodedist.Download(ctx, version, target.archiveName(version))
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "goreleaser-node-*")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, target.hostBinaryName())

	if target.IsWindows() {
		if err := os.Rename(archive, bin); err != nil {
			return "", err
		}
	} else {
		if err := extractFromTarGz(archive, target.tarEntry(version), bin); err != nil {
			return "", err
		}
		_ = os.Remove(archive)
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return "", err
	}
	return bin, nil
}

// extractFromTarGz finds entry inside the gzipped tar at archivePath
// and writes it atomically to dst. We never join tar entry names onto
// dst — only the single, fully qualified entry the caller asks for is
// extracted — so there is no zip-slip surface.
func extractFromTarGz(archivePath, entry, dst string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h.Name != entry {
			continue
		}
		tmp, err := os.CreateTemp(filepath.Dir(dst), ".extract-*")
		if err != nil {
			return err
		}
		tmpName := tmp.Name()
		if _, err := io.Copy(tmp, tr); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			return err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
		if err := os.Rename(tmpName, dst); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
		return nil
	}
	return fmt.Errorf("node: %s not found in %s", entry, archivePath)
}
