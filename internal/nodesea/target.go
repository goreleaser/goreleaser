package nodesea

import (
	"fmt"
	"path"
	"strings"
)

// Target is a Node.js distribution target as named by nodejs.org/dist
// (e.g. "linux-x64", "darwin-arm64", "win-x64"). It is the bridge
// between goreleaser's Go-style GOOS/GOARCH and the nodejs.org URL
// layout.
type Target string

// String returns the raw target identifier.
func (t Target) String() string { return string(t) }

// Goos returns the Go GOOS value matching the target.
func (t Target) Goos() string {
	prefix, _, _ := strings.Cut(string(t), "-")
	if prefix == "win" {
		return "windows"
	}
	return prefix
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

// tarEntry returns the path of the node binary inside the published
// tar.gz archive for the given version. Only meaningful for non-windows
// targets.
func (t Target) tarEntry(version string) string {
	return fmt.Sprintf("node-%s-%s/bin/node", version, t)
}
