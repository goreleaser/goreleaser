package nodesea

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
)

// buildToolEnv is the environment variable that overrides the Node.js
// binary used to drive `node --build-sea`. When set, it is consulted
// first by BuildToolNode and must satisfy probeBuildSEACapable.
const buildToolEnv = "GORELEASER_NODE_BUILD_TOOL"

// buildToolNodeVersion is the host-platform Node release auto-downloaded
// when neither the env override nor the host `node` on PATH satisfies
// the build-sea capability probe. Bumped per goreleaser release.
const buildToolNodeVersion = "v25.9.0"

// errBuildSEAUnsupported is wrapped by probeBuildSEACapable when a Node
// binary cannot drive `--build-sea` (either the option does not exist
// or `process.config.variables.node_use_lief` is not true).
var errBuildSEAUnsupported = errors.New("nodesea: node binary lacks --build-sea LIEF support")

// runProbe executes the build-sea capability probe against nodePath. It
// is a package-level variable so tests can stub it without requiring a
// real Node binary on disk.
//
//nolint:gochecknoglobals
var runProbe = func(ctx context.Context, nodePath string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, nodePath, "-p", "process.config.variables.node_use_lief")
	return cmd.CombinedOutput()
}

// BuildToolNode resolves an absolute path to a Node binary that can
// drive `node --build-sea sea-config.json`. The lookup order is:
//
//  1. $GORELEASER_NODE_BUILD_TOOL — when set, the named binary must
//     satisfy the probe; otherwise BuildToolNode returns an error.
//  2. The first `node` on PATH — used iff it satisfies the probe; an
//     unsuitable PATH binary is silently skipped.
//  3. Auto-download into <CacheDir>/buildtool/<buildToolNodeVersion>/.
//
// The returned path is guaranteed to satisfy probeBuildSEACapable.
func BuildToolNode(ctx context.Context) (string, error) {
	if envPath := os.Getenv(buildToolEnv); envPath != "" {
		resolved, err := exec.LookPath(envPath)
		if err != nil {
			return "", fmt.Errorf("nodesea: %s=%q: %w", buildToolEnv, envPath, err)
		}
		if err := probeBuildSEACapable(ctx, resolved); err != nil {
			return "", fmt.Errorf("nodesea: %s=%q: %w", buildToolEnv, envPath, err)
		}
		return resolved, nil
	}
	if pathNode, err := exec.LookPath("node"); err == nil {
		if err := probeBuildSEACapable(ctx, pathNode); err == nil {
			return pathNode, nil
		}
	}
	return downloadBuildToolNode(ctx)
}

// probeBuildSEACapable returns nil iff nodePath can drive `--build-sea`
// with LIEF backing. The check matches the gate Node's own test suite
// uses (see test/common/sea.js#skipIfBuildSEAIsNotSupported): we read
// process.config.variables.node_use_lief and require the literal value
// `true`. Anything else (`false`, `undefined`, exec failure) reports
// errBuildSEAUnsupported.
func probeBuildSEACapable(ctx context.Context, nodePath string) error {
	out, err := runProbe(ctx, nodePath)
	got := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("%w: probe %s: %w (output: %q)",
			errBuildSEAUnsupported, nodePath, err, got)
	}
	if got != "true" {
		return fmt.Errorf("%w: %s reported node_use_lief=%q (want %q)",
			errBuildSEAUnsupported, nodePath, got, "true")
	}
	return nil
}

// downloadBuildToolNode fetches the host-platform Node release at
// buildToolNodeVersion into <cache>/buildtool/<version>/<target>/ and
// confirms it satisfies the capability probe before returning.
func downloadBuildToolNode(ctx context.Context) (string, error) {
	cacheDir, err := nodedist.CacheDir()
	if err != nil {
		return "", err
	}
	target := nodedist.Target(currentTarget())
	btDir := filepath.Join(cacheDir, "buildtool")
	nodePath, err := nodedist.Download(ctx, btDir, buildToolNodeVersion, target)
	if err != nil {
		return "", fmt.Errorf("nodesea: download build-tool node %s: %w", buildToolNodeVersion, err)
	}
	if err := probeBuildSEACapable(ctx, nodePath); err != nil {
		return "", err
	}
	return nodePath, nil
}

// currentTarget returns the nodejs.org/dist target identifier matching
// the machine running goreleaser, e.g. "linux-x64" or "darwin-arm64".
// It mirrors the helper in internal/builders/node so this package has
// no dependency cycle on the builder.
func currentTarget() string {
	osName := runtime.GOOS
	if osName == "windows" {
		osName = "win"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return osName + "-" + arch
}
