package node

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
)

// Supported nodejs.org/dist target identifiers. Kept in lockstep with the
// targets actually published under https://nodejs.org/dist/<version>/.
//
//go:embed targets.txt
var allTargetsBts []byte

//nolint:gochecknoglobals
var (
	allTargets  []string
	targetsOnce sync.Once
)

// Target represents a build target.
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
func (t Target) String() string {
	return t.Target
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	if !isValid(target) {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}
	parts := strings.Split(target, "-")
	return Target{
		Target: target,
		Os:     parts[0],
		Arch:   parts[1],
	}, nil
}

func convertToGoarch(s string) string {
	switch s {
	case "x64":
		return "amd64"
	case "armv7l":
		// Node ships a single 32-bit ARMv7 hard-float build under
		// this name; map it to Go's GOARCH=arm with implicit GOARM=7.
		return "arm"
	default:
		return s
	}
}

func convertToGoos(s string) string {
	switch s {
	case "win":
		return "windows"
	default:
		return s
	}
}

// goarmFor returns the GOARM value implied by the nodejs.org arch
// component, or "" when no GOARM applies. Node's only 32-bit ARM
// publication is the ARMv7 hard-float build, so armv7l implies "7".
func goarmFor(arch string) string {
	if arch == "armv7l" {
		return "7"
	}
	return ""
}

func isValid(target string) bool {
	targetsOnce.Do(func() {
		for t := range strings.SplitSeq(string(allTargetsBts), "\n") {
			if t = strings.TrimSpace(t); t != "" {
				allTargets = append(allTargets, t)
			}
		}
	})
	return slices.Contains(allTargets, target)
}

func defaultTargets() []string {
	return slices.Clone(append([]string(nil), []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"win-arm64",
		"win-x64",
	}...))
}
