package nodedist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/goreleaser/goreleaser/v2/internal/retryx"
)

// Download fetches https://nodejs.org/dist/<version>/<archiveName>
// into a fresh temp file, verifies its SHA-256 against the embedded
// release index, and returns the local file path. Each call hits the
// network — there is no on-disk cache. The temp file lives under
// os.TempDir() and is reaped by the OS on its usual schedule.
//
// The caller owns extraction (for tar.gz archives) and any further
// mutation (e.g. stripping a code signature) before use.
func Download(ctx context.Context, version, archiveName string) (string, error) {
	expected, err := lookupSHA(version, archiveName)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "goreleaser-nodedist-*")
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()

	url := fmt.Sprintf("%s/%s/%s", distBaseURL, version, archiveName)
	hash, err := downloadTo(ctx, url, tmp)
	_ = tmp.Close()
	if err != nil {
		_ = os.Remove(tmpName)
		return "", err
	}
	if expected != hash {
		_ = os.Remove(tmpName)
		return "", fmt.Errorf("nodedist: SHA-256 mismatch for %s: expected %s, got %s", url, expected, hash)
	}
	return tmpName, nil
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
