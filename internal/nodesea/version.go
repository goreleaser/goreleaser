package nodesea

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// VersionSource describes where a Node.js version came from. It is purely
// informational and used for log/error messages.
type VersionSource string

const (
	// VersionSourceExplicit means the version was passed in directly by the
	// caller (e.g. via the build configuration).
	VersionSourceExplicit VersionSource = "explicit"
	// VersionSourceEnginesNode means the version was resolved from
	// `package.json`'s `engines.node` field.
	VersionSourceEnginesNode VersionSource = "package.json engines.node"
	// VersionSourceNvmrc means the version was read from a `.nvmrc` file.
	VersionSourceNvmrc VersionSource = ".nvmrc"
	// VersionSourceNodeVersion means the version was read from a
	// `.node-version` file.
	VersionSourceNodeVersion VersionSource = ".node-version"
)

// ErrNoVersion is returned by ResolveVersion when no version can be
// determined from any of the supported sources.
var ErrNoVersion = errors.New("nodesea: could not resolve a Node.js version; set node_version in the build, or add engines.node to package.json, .nvmrc or .node-version")

// indexEntry mirrors one record in https://nodejs.org/dist/index.json,
// which lists every published Node.js release.
type indexEntry struct {
	Version string `json:"version"`
	LTS     any    `json:"lts"`
}

// indexFetcher returns the parsed nodejs.org release index. It is a
// package-level variable so tests can stub it without hitting the
// network.
//
//nolint:gochecknoglobals
var indexFetcher = fetchNodeIndex

func fetchNodeIndex(ctx context.Context) ([]indexEntry, error) {
	const u = "https://nodejs.org/dist/index.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nodesea: fetch %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nodesea: fetch %s: unexpected status %s", u, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var entries []indexEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("nodesea: decode index.json: %w", err)
	}
	return entries, nil
}

// ResolveVersion picks a Node.js version, in order:
//
//  1. explicit (caller-supplied string, e.g. from the build config);
//  2. `engines.node` in package.json;
//  3. `.nvmrc`;
//  4. `.node-version`.
//
// Explicit and file-based values may be either an exact version
// (`v22.10.0`, `22.10.0`) or a semver range. Ranges are resolved to the
// highest matching release published on nodejs.org/dist. The returned
// version always carries a leading `v` to match nodejs.org URL paths.
func ResolveVersion(ctx context.Context, dir, explicit string) (string, VersionSource, error) {
	candidates := []struct {
		raw    string
		source VersionSource
	}{
		{explicit, VersionSourceExplicit},
	}

	if pkgRange, err := readEnginesNode(filepath.Join(dir, "package.json")); err == nil && pkgRange != "" {
		candidates = append(candidates, struct {
			raw    string
			source VersionSource
		}{pkgRange, VersionSourceEnginesNode})
	}
	for _, fname := range []string{".nvmrc", ".node-version"} {
		raw, err := readVersionFile(filepath.Join(dir, fname))
		if err == nil && raw != "" {
			source := VersionSourceNvmrc
			if fname == ".node-version" {
				source = VersionSourceNodeVersion
			}
			candidates = append(candidates, struct {
				raw    string
				source VersionSource
			}{raw, source})
		}
	}

	for _, c := range candidates {
		if c.raw == "" {
			continue
		}
		ver, err := resolveVersionString(ctx, c.raw)
		if err != nil {
			return "", c.source, fmt.Errorf("nodesea: resolve %s value %q: %w", c.source, c.raw, err)
		}
		return ver, c.source, nil
	}
	return "", "", ErrNoVersion
}

// resolveVersionString turns a user-supplied version specifier into a
// concrete `vX.Y.Z` release that exists on nodejs.org.
func resolveVersionString(ctx context.Context, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")

	// Exact pinned version: don't touch the network.
	if v, err := semver.StrictNewVersion(raw); err == nil {
		return "v" + v.String(), nil
	}

	constraint, err := semver.NewConstraint(raw)
	if err != nil {
		return "", fmt.Errorf("invalid semver constraint: %w", err)
	}

	entries, err := indexFetcher(ctx)
	if err != nil {
		return "", err
	}

	matched := make([]*semver.Version, 0, len(entries))
	for _, e := range entries {
		v, err := semver.NewVersion(strings.TrimPrefix(e.Version, "v"))
		if err != nil {
			continue
		}
		if v.Prerelease() != "" {
			continue
		}
		if constraint.Check(v) {
			matched = append(matched, v)
		}
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no published Node.js release satisfies %q", raw)
	}
	sort.Sort(semver.Collection(matched))
	return "v" + matched[len(matched)-1].String(), nil
}

// readEnginesNode extracts the `engines.node` value from a package.json
// file. It returns "" (and a nil error) when the file does not exist or
// the field is missing.
func readEnginesNode(path string) (string, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal(bts, &pkg); err != nil {
		return "", err
	}
	return strings.TrimSpace(pkg.Engines.Node), nil
}

// readVersionFile reads the first non-empty, non-comment line of a
// version file (.nvmrc / .node-version). It returns "" when the file does
// not exist.
func readVersionFile(path string) (string, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	for line := range strings.SplitSeq(string(bts), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, nil
	}
	return "", nil
}
