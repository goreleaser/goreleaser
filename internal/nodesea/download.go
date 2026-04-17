package nodesea

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
	"path"
	"path/filepath"
	"strings"
)

// Target represents a Node.js distribution target as named by
// nodejs.org/dist (e.g. "linux-x64", "darwin-arm64", "win-x64").
type Target string

// Goos returns the Go GOOS value matching the target.
func (t Target) Goos() string {
	prefix, _, _ := strings.Cut(string(t), "-")
	switch prefix {
	case "win":
		return "windows"
	default:
		return prefix
	}
}

// Goarch returns the Go GOARCH value matching the target.
func (t Target) Goarch() string {
	_, arch, _ := strings.Cut(string(t), "-")
	switch arch {
	case "x64":
		return "amd64"
	default:
		return arch
	}
}

// IsWindows reports whether the target is a windows distribution.
func (t Target) IsWindows() bool { return strings.HasPrefix(string(t), "win-") }

// archiveName returns the file name nodejs.org publishes for this target
// under https://nodejs.org/dist/<version>/.
func (t Target) archiveName(version string) string {
	if t.IsWindows() {
		return path.Join(string(t), "node.exe")
	}
	return fmt.Sprintf("node-%s-%s.tar.gz", version, t)
}

// hostBinaryName returns the basename of the Node.js executable for this
// target ("node" or "node.exe").
func (t Target) hostBinaryName() string {
	if t.IsWindows() {
		return "node.exe"
	}
	return "node"
}

// distBaseURL is the prefix every nodejs.org/dist URL uses. It is a
// package-level variable so tests can point it at a stub server.
//
//nolint:gochecknoglobals
var distBaseURL = "https://nodejs.org/dist"

// httpClient is the HTTP client used for nodejs.org downloads. Tests may
// override it; production uses http.DefaultClient.
//
//nolint:gochecknoglobals
var httpClient = http.DefaultClient

// CacheDir returns the directory used to cache downloaded Node.js host
// binaries. It honours XDG_CACHE_HOME and falls back to ~/.cache. When
// the user cache directory cannot be determined (very unusual), it
// returns an empty string and an error, leaving the caller to pick a
// fallback.
func CacheDir() (string, error) {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "goreleaser", "node"), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "goreleaser", "node"), nil
}

// downloadHost fetches and extracts the Node.js host binary for the given
// target and version into cacheDir. The returned path points at the
// extracted host binary. If a cached copy already exists it is returned
// without touching the network.
//
// The caller is responsible for any further mutation (e.g. stripping a
// code signature) before injecting a SEA blob.
func downloadHost(ctx context.Context, cacheDir, version string, target Target) (string, error) {
	hostDir := filepath.Join(cacheDir, version, string(target))
	hostPath := filepath.Join(hostDir, target.hostBinaryName())

	if _, err := os.Stat(hostPath); err == nil {
		return hostPath, nil
	}

	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return "", err
	}

	archiveURL := fmt.Sprintf("%s/%s/%s", distBaseURL, version, target.archiveName(version))
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

	expected, err := fetchExpectedSHA(ctx, version, target.archiveName(version))
	if err != nil {
		return "", err
	}
	if expected != hash {
		return "", fmt.Errorf("nodesea: SHA-256 mismatch for %s: expected %s, got %s", archiveURL, expected, hash)
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

// downloadTo streams an HTTP GET into dst, returning the SHA-256 of the
// downloaded bytes as a lower-case hex string.
func downloadTo(ctx context.Context, url string, dst io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nodesea: download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nodesea: download %s: unexpected status %s", url, resp.Status)
	}

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h), resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// fetchExpectedSHA fetches SHASUMS256.txt for the release and returns the
// SHA-256 line matching the supplied archive file name.
func fetchExpectedSHA(ctx context.Context, version, archiveName string) (string, error) {
	url := fmt.Sprintf("%s/%s/SHASUMS256.txt", distBaseURL, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nodesea: download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nodesea: download %s: unexpected status %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// fields[1] is "node-vX.Y.Z-linux-x64.tar.gz" or
		// "win-x64/node.exe".
		if fields[1] == archiveName {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("nodesea: %s not present in %s", archiveName, url)
}

// extractNodeFromTarGz finds the bin/node entry inside the released
// nodejs tarball and writes it to dst.
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
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("nodesea: %s not found in %s", want, archivePath)
}
