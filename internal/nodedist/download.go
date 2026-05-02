package nodedist

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/v2/internal/retryx"
)

// Download fetches and extracts the Node.js host binary for the given
// target and version into a fresh temp directory and returns the
// absolute path to it. Each call hits the network — there is no
// on-disk cache. The temp directory lives under os.TempDir() and is
// reaped by the OS on its usual schedule.
//
// The expected SHA-256 is read from the embedded release index
// (releases.json); only versions present there can be downloaded.
//
// The caller is responsible for any further mutation (e.g. stripping
// a code signature) before injecting a SEA blob.
func Download(ctx context.Context, version string, target Target) (string, error) {
	archiveName := target.ArchiveName(version)
	expected, err := LookupSHA(version, archiveName)
	if err != nil {
		return "", err
	}

	hostDir, err := os.MkdirTemp("", "goreleaser-node-*")
	if err != nil {
		return "", err
	}
	hostPath := filepath.Join(hostDir, target.HostBinaryName())

	archiveURL := fmt.Sprintf("%s/%s/%s", distBaseURL, version, archiveName)
	tmp, err := os.CreateTemp(hostDir, "download-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hash, err := downloadTo(ctx, archiveURL, tmp)
	_ = tmp.Close()
	if err != nil {
		return "", err
	}

	if expected != hash {
		return "", fmt.Errorf("nodedist: SHA-256 mismatch for %s: expected %s, got %s", archiveURL, expected, hash)
	}

	if target.IsWindows() {
		if err := os.Rename(tmpName, hostPath); err != nil {
			return "", err
		}
	} else if err := extractNodeFromTarGz(tmpName, version, target, hostPath); err != nil {
		return "", err
	}

	if err := os.Chmod(hostPath, 0o755); err != nil {
		return "", err
	}
	return hostPath, nil
}

// downloadTo streams an HTTP GET into dst, returning the SHA-256 of
// the downloaded bytes as a lower-case hex string. Transient failures
// (5xx, 429, network errors) are retried per defaultRetry; on each
// attempt dst is rewound and truncated so a partially downloaded body
// from a failed attempt does not pollute the next one.
func downloadTo(ctx context.Context, url string, dst *os.File) (string, error) {
	var hash string
	err := retryx.Do(ctx, defaultRetry, func() error {
		if _, err := dst.Seek(0, io.SeekStart); err != nil {
			return retryx.Unrecoverable(err)
		}
		if err := dst.Truncate(0); err != nil {
			return retryx.Unrecoverable(err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return retryx.HTTP(fmt.Errorf("nodedist: download %s: %w", url, err), resp)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return retryx.HTTP(fmt.Errorf("nodedist: download %s: unexpected status %s", url, resp.Status), resp)
		}
		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(dst, h), resp.Body); err != nil {
			return retryx.HTTP(err, resp)
		}
		hash = hex.EncodeToString(h.Sum(nil))
		return nil
	}, retryx.IsRetriable)
	return hash, err
}

// extractNodeFromTarGz finds the bin/node entry inside the released
// nodejs tarball and writes it to dst.
//
// We never join tar entry names onto dst — only the single, fully
// qualified entry we ask for is extracted, and the destination is the
// caller-controlled dst — so there is no zip-slip surface.
//
// Extraction is atomic: data is streamed into a sibling tempfile and
// renamed over dst on success, so an interrupted extract never leaves
// a truncated file at the canonical cache path.
func extractNodeFromTarGz(archivePath, version string, target Target, dst string) error {
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

	want := fmt.Sprintf("node-%s-%s/bin/node", version, target)
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if h.Name != want {
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
	return fmt.Errorf("nodedist: %s not found in %s", want, archivePath)
}
