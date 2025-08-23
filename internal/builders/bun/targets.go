package bun

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
)

// https://bun.sh/docs/bundler/executables
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
	Abi    string
	Type   string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
		"Abi":        t.Abi,
		"Type":       t.Type,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	target = strings.TrimPrefix(target, "bun-")
	parts := strings.Split(target, "-")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	t := Target{
		Target: "bun-" + target,
		Os:     parts[0],
		Arch:   parts[1],
	}

	if len(parts) > 2 {
		switch p := parts[2]; p {
		case "musl":
			t.Abi = p
			if len(parts) > 3 {
				t.Type = parts[3]
			}
		default:
			t.Type = p
		}
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

	return slices.Contains(allTargets, target) ||
		slices.Contains(allTargets, "bun-"+target)
}

func defaultTargets() []string {
	return []string{
		"linux-x64-modern",
		"linux-arm64",
		"darwin-x64",
		"darwin-arm64",
		"windows-x64-modern",
	}
}
