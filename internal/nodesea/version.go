package nodesea

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
)

// VersionSource describes where a Node.js version came from. It is purely
// informational and used for log/error messages.
type VersionSource string

const (
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
var ErrNoVersion = errors.New("nodesea: could not resolve a Node.js version; add engines.node to package.json (or .nvmrc / .node-version)")

// ResolveVersion picks a Node.js version from the user's project
// directory, in order:
//
//  1. `engines.node` in package.json;
//  2. `.nvmrc`;
//  3. `.node-version`.
//
// File-based values may be either an exact version (`v22.10.0`,
// `22.10.0`) or a semver range. Ranges are resolved to the highest
// matching release published on nodejs.org/dist. The returned version
// always carries a leading `v` to match nodejs.org URL paths.
func ResolveVersion(dir string) (string, VersionSource, error) {
	var candidates []struct {
		raw    string
		source VersionSource
	}

	pkg, err := packagejson.OpenOrEmpty(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", "", fmt.Errorf("nodesea: read package.json: %w", err)
	}
	if r := pkg.Engines.NodeRange(); r != "" {
		candidates = append(candidates, struct {
			raw    string
			source VersionSource
		}{r, VersionSourceEnginesNode})
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
		ver, err := resolveVersionString(c.raw)
		if err != nil {
			return "", c.source, fmt.Errorf("nodesea: resolve %s value %q: %w", c.source, c.raw, err)
		}
		return ver, c.source, nil
	}
	return "", "", ErrNoVersion
}

// resolveVersionString turns a user-supplied version specifier into a
// concrete `vX.Y.Z` release present in the embedded nodedist index.
func resolveVersionString(raw string) (string, error) {
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

	entries := nodedist.Releases()
	matched := make([]*semver.Version, 0, len(entries))
	for verStr := range entries {
		v, err := semver.NewVersion(strings.TrimPrefix(verStr, "v"))
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
