package rust

import (
	"slices"
	"strings"
	"sync"

	_ "embed"

	"github.com/goreleaser/goreleaser/v2/internal/builders/buildtarget"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
)

// tier 1 and tier 2
// aarch64-pc-windows-gnullvm is the only tier 3 target added
// https://doc.rust-lang.org/rustc/platform-support.html
var (
	//go:embed all_targets.txt
	allTargetsBts []byte
	allTargets    []string
	targetsOnce   sync.Once
)

const (
	keyVendor = "Vendor"
	keyAbi    = "Abi"
	keyLibc   = "Libc"
)

// Target is a Rust build target.
type Target struct {
	// The Rust formatted target (arch-vendor-os-env).
	Target string
	Os     string
	Arch   string
	Arm    string
	Vendor string
	Abi    string
	Libc   string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
		tmpl.KeyArm:  t.Arm,
		keyAbi:       t.Abi,
		keyVendor:    t.Vendor,
		keyLibc:      t.Libc,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

// clean returns the target without any gnu version suffix.
// this is used by cargo-zigbuild internally.
func (t Target) clean() string {
	if clean, ok := stripGlibcVersion(t.Target); ok {
		return clean
	}
	return t.Target
}

// stripGlibcVersion removes a glibc version suffix (e.g., ".2.17") from
// gnu-based targets.
// Returns the base target and true if a suffix was stripped.
func stripGlibcVersion(target string) (string, bool) {
	prefix, _, ok := strings.Cut(target, ".")
	if !ok {
		return target, false
	}
	// only gnu-based ABIs use glibc (not gnullvm which is Windows/LLVM)
	_, abi, found := strings.CutLast(prefix, "-")
	if !found {
		return target, false
	}
	if strings.HasPrefix(abi, "gnu") && abi != "gnullvm" {
		return prefix, true
	}
	return target, false
}

func convertToGoarch(s string) string {
	return buildtarget.Goarch(s)
}

func isValid(target string) bool {
	targetsOnce.Do(func() {
		allTargets = slices.DeleteFunc(strings.Split(string(allTargetsBts), "\n"), func(s string) bool {
			return s == ""
		})
	})
	if clean, ok := stripGlibcVersion(target); ok {
		return slices.Contains(allTargets, clean)
	}
	return slices.Contains(allTargets, target)
}

func defaultTargets() []string {
	return []string{
		"x86_64-unknown-linux-gnu",
		"x86_64-apple-darwin",
		"x86_64-pc-windows-gnu",
		"aarch64-unknown-linux-gnu",
		"aarch64-apple-darwin",
	}
}

// Get ARM version for `(artifact.Artifact).Goarm`
func getArmVersion(s string) (string, bool) {
	// Values sourced from https://go.dev/wiki/GoArm in combination with
	// https://doc.rust-lang.org/nightly/rustc/platform-support.html
	ss, ok := map[string]string{
		"arm":     "6",
		"armv5te": "5",
		"armv7":   "7",
		"armv7a":  "7",
		"armv7r":  "7",
	}[s]
	return ss, ok
}
