// Package nodedist is a small client for the official Node.js
// distribution at https://nodejs.org/dist. It exposes the per-target
// archive layout, the release index, a SHA-256 verifying downloader
// and a cache directory shared with goreleaser's other Node.js code.
//
// The package is intentionally infrastructure-only: it knows nothing
// about Single Executable Applications, version resolution from
// project files, or build orchestration. Callers (typically the
// nodesea package) layer those concerns on top.
package nodedist

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
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
	if arch == "x64" {
		return "amd64"
	}
	return arch
}

// IsWindows reports whether the target is a windows distribution.
func (t Target) IsWindows() bool { return strings.HasPrefix(string(t), "win-") }

// ArchiveName returns the file name nodejs.org publishes for this
// target under https://nodejs.org/dist/<version>/.
func (t Target) ArchiveName(version string) string {
	if t.IsWindows() {
		return path.Join(string(t), "node.exe")
	}
	return fmt.Sprintf("node-%s-%s.tar.gz", version, t)
}

// HostBinaryName returns the basename of the Node.js executable for
// this target ("node" or "node.exe").
func (t Target) HostBinaryName() string {
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

// httpClient is the HTTP client used for nodejs.org downloads. Tests
// may override it; production uses http.DefaultClient.
//
//nolint:gochecknoglobals
var httpClient = http.DefaultClient

// defaultRetry is the retry policy applied to every nodejs.org HTTP
// fetch (release index, archive, SHASUMS). It mirrors what the rest
// of goreleaser uses for transient server errors but is fixed at
// package scope rather than threaded from ctx.Config.Retry to avoid
// leaking a goreleaser context dependency into nodedist — these
// downloads happen behind the cache, so per-project tuning is rarely
// interesting.
//
//nolint:gochecknoglobals
var defaultRetry = config.Retry{
	Attempts: 4,
	Delay:    time.Second,
	MaxDelay: 30 * time.Second,
}
