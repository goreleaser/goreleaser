package node

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
)

// supportedTargets is the canonical list of nodejs.org/dist target
// identifiers the builder accepts. Kept in lockstep with the targets
// actually published under https://nodejs.org/dist/<version>/.
//
//nolint:gochecknoglobals
var supportedTargets = []string{
	"darwin-arm64",
	"darwin-x64",
	"linux-arm64",
	"linux-x64",
	"win-arm64",
	"win-x64",
}

// Target represents a build target (a nodejs.org/dist identifier such
// as "linux-x64" plus the parsed Os/Arch components).
type Target struct {
	Target string
	Os     string
	Arch   string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string { return t.Target }

// Goos returns the Go GOOS value matching the target.
func (t Target) Goos() string {
	if t.Os == "win" {
		return "windows"
	}
	return t.Os
}

// Goarch returns the Go GOARCH value matching the target.
func (t Target) Goarch() string {
	if t.Arch == "x64" {
		return "amd64"
	}
	return t.Arch
}

// IsWindows reports whether the target is a windows distribution.
func (t Target) IsWindows() bool { return t.Os == "win" }

// tarEntry returns the path of the node binary inside the published
// tar.gz archive for the given version. Only meaningful for non-windows
// targets.
func (t Target) tarEntry(version string) string {
	return fmt.Sprintf("node-%s-%s/bin/node", version, t.Target)
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	t, ok := parseTarget(target)
	if !ok {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}
	return t, nil
}

// parseTarget splits a nodejs.org/dist identifier and reports whether
// it's in the supported set.
func parseTarget(target string) (Target, bool) {
	if !slices.Contains(supportedTargets, target) {
		return Target{}, false
	}
	os, arch, _ := strings.Cut(target, "-")
	return Target{Target: target, Os: os, Arch: arch}, true
}

func defaultTargets() []string { return slices.Clone(supportedTargets) }
