package node

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
)

// errNoVersion is returned by resolveVersion when no version can be
// determined from package.json.
var errNoVersion = errors.New("node: could not resolve a Node.js version; add engines.node to package.json")

const minNodeSEAVersion = "25.5.0"

// ensureNode resolves the node version, downloads it, and returns its path.
func ensureNode(ctx context.Context, dir, target string) (string, error) {
	version, err := resolveVersion(dir)
	if err != nil {
		return "", fmt.Errorf("node: resolve node version: %w", err)
	}

	return downloadHostBinary(ctx, version, target)
}

// resolveVersion picks a Node.js version from `engines.node` in the
// project's package.json. Either an exact version (`v25.5.0`,
// `25.5.0`) or a semver range (`>=25.5 <26`, `^25`) is accepted.
// Ranges are resolved to the highest matching release in the embedded
// nodedist index. The returned version always carries a leading `v`
// to match nodejs.org URL paths.
func resolveVersion(dir string) (string, error) {
	pkg, err := packagejson.Open(filepath.Join(dir, "package.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("node: read package.json: %w", err)
	}
	raw := strings.TrimSpace(pkg.Engines["node"])
	if raw == "" {
		return "", errNoVersion
	}
	ver, err := resolveVersionString(raw)
	if err != nil {
		return "", fmt.Errorf("node: resolve package.json engines.node value %q: %w", raw, err)
	}
	return ver, nil
}

// resolveVersionString turns a user-supplied version specifier into a
// concrete `vX.Y.Z` release present in the embedded nodedist index.
func resolveVersionString(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")

	// Exact pinned version: don't touch the network.
	if v, err := semver.StrictNewVersion(raw); err == nil {
		if err := validateNodeSEAVersion(v); err != nil {
			return "", err
		}
		return "v" + v.String(), nil
	}

	constraint, err := semver.NewConstraint(raw)
	if err != nil {
		return "", fmt.Errorf("invalid semver constraint: %w", err)
	}

	var found *semver.Version
	for verStr := range nodedist.Releases() {
		v, err := semver.NewVersion(verStr)
		if err != nil {
			continue
		}
		if constraint.Check(v) {
			if found == nil || v.GreaterThan(found) {
				found = v
			}
		}
	}
	if found == nil {
		return "", fmt.Errorf("no published Node.js release satisfies %q", raw)
	}
	if err := validateNodeSEAVersion(found); err != nil {
		return "", err
	}
	return "v" + found.String(), nil
}

func validateNodeSEAVersion(v *semver.Version) error {
	minV := semver.MustParse(minNodeSEAVersion)
	if v.LessThan(minV) {
		return fmt.Errorf("node.js SEA requires Node.js >= v%s, got v%s", minV, v)
	}
	return nil
}

func checkHostNodeVersion(ctx context.Context, tool string, env []string) error {
	cmd := exec.CommandContext(ctx, tool, "--version") //nolint:gosec
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("node: check host node version: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return fmt.Errorf("node: check host node version: empty output from %s", tool)
	}
	v, err := semver.NewVersion(strings.TrimPrefix(fields[0], "v"))
	if err != nil {
		return fmt.Errorf("node: parse host node version %q: %w", strings.TrimSpace(string(out)), err)
	}
	if err := validateNodeSEAVersion(v); err != nil {
		return fmt.Errorf("node: host node %s does not support SEA: %w", tool, err)
	}
	return nil
}

// downloadHostBinary fetches the per-target Node.js host binary for
// (version, target) and returns its absolute path. tar.gz archives
// are extracted; bare windows .exe archives are used as-is. The
// returned path lives under a fresh temp directory.
func downloadHostBinary(ctx context.Context, version, target string) (string, error) {
	log.WithField("version", version).
		WithField("target", target).
		Info("downloading")

	isWin := strings.HasPrefix(target, "win-")
	archiveFile := fmt.Sprintf("node-%s-%s.tar.gz", version, target)
	binName := "node"
	if isWin {
		archiveFile = path.Join(target, "node.exe")
		binName = "node.exe"
	}

	archive, err := nodedist.Download(ctx, version, archiveFile)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "goreleaser-node-*")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, binName)

	if isWin {
		if err := os.Rename(archive, bin); err != nil {
			return "", err
		}
	} else {
		entry := fmt.Sprintf("node-%s-%s/bin/node", version, target)
		if err := extractFromTarGz(archive, entry, bin); err != nil {
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
