package deno

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
)

const keyAbi = "Abi"

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
	Vendor string
	Abi    string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
		"Vendor":     t.Vendor,
		keyAbi:       t.Abi,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

// Parse implements build.Builder.
func (b *Builder) Parse(target string) (api.Target, error) {
	parts := strings.Split(target, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("%s is not a valid build target", target)
	}

	t := Target{
		Target: target,
		Os:     parts[2],
		Arch:   parts[0],
		Vendor: parts[1],
	}

	if len(parts) > 3 {
		t.Abi = parts[3]
	}

	return t, nil
}

func convertToGoarch(s string) string {
	ss, ok := map[string]string{
		"aarch64": "arm64",
		"x86_64":  "amd64",
	}[s]
	if ok {
		return ss
	}
	return s
}

func isValid(target string) bool {
	targetsOnce.Do(func() {
		allTargets = strings.Split(string(allTargetsBts), "\n")
	})

	return slices.Contains(allTargets, target)
}

func defaultTargets() []string {
	return []string{
		"x86_64-pc-windows-msvc",
		"x86_64-apple-darwin",
		"aarch64-apple-darwin",
		"x86_64-unknown-linux-gnu",
		"aarch64-unknown-linux-gnu",
	}
}
