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

// https://docs.deno.com/runtime/reference/cli/compile/#supported-targets
var (
	//go:embed targets.txt
	allTargetsBts []byte
	allTargets    []string
	targetsOnce   sync.Once
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
	parts := strings.Split(target, "-")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	t := Target{
		Target: target,
		Os:     parts[0],
		Arch: parts[1],
	}

	return t, nil
}

func convertToGoarch(s string) string {
	switch s {
	case "x64":
		return "amd64"
	default:
		return s
	}
}

func isValid(target string) bool {
	targetsOnce.Do(func() {
		allTargets = strings.Split(string(allTargetsBts), "\n")
	})

	return slices.Contains(allTargets, target)
}

func defaultTargets() []string {
	return []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"win-arm64",
		"win-x64",
	}
}
