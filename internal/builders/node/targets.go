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

func isValid(target string) bool {
	return slices.Contains(supportedTargets, target)
}

func defaultTargets() []string {
	return slices.Clone(supportedTargets)
}
