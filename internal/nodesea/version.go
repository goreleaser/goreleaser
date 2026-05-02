package nodesea

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
)

// errNoVersion is returned by ResolveVersion when no version can be
// determined from package.json.
var errNoVersion = errors.New("nodesea: could not resolve a Node.js version; add engines.node to package.json")

// ResolveVersion picks a Node.js version from `engines.node` in the
// project's package.json. Either an exact version (`v25.5.0`,
// `25.5.0`) or a semver range (`>=25.5 <26`, `^25`) is accepted.
// Ranges are resolved to the highest matching release in the embedded
// nodedist index. The returned version always carries a leading `v`
// to match nodejs.org URL paths.
func ResolveVersion(dir string) (string, error) {
	pkg, err := packagejson.OpenOrEmpty(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("nodesea: read package.json: %w", err)
	}
	raw := pkg.Engines.NodeRange()
	if raw == "" {
		return "", errNoVersion
	}
	ver, err := resolveVersionString(raw)
	if err != nil {
		return "", fmt.Errorf("nodesea: resolve package.json engines.node value %q: %w", raw, err)
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
		return "v" + v.String(), nil
	}

	constraint, err := semver.NewConstraint(raw)
	if err != nil {
		return "", fmt.Errorf("invalid semver constraint: %w", err)
	}

	entries := nodedist.Releases()
	matched := make([]*semver.Version, 0, len(entries))
	for verStr := range entries {
		v, err := semver.NewVersion(verStr)
		if err != nil {
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
