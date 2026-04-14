package rust

import (
	"slices"
	"strings"
	"sync"

	_ "embed"

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
	Vendor string
	Abi    string
	Libc   string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
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
	lastDash := strings.LastIndex(prefix, "-")
	if lastDash == -1 {
		return target, false
	}
	abi := prefix[lastDash+1:]
	if strings.HasPrefix(abi, "gnu") && abi != "gnullvm" {
		return prefix, true
	}
	return target, false
}

func convertToGoarch(s string) string {
	ss, ok := map[string]string{
		"aarch64":     "arm64",
		"x86_64":      "amd64",
		"i686":        "386",
		"i586":        "386",
		"i386":        "386",
		"powerpc":     "ppc",
		"powerpc64":   "ppc64",
		"powerpc64le": "ppc64le",
		"riscv64":     "riscv64",
		"s390x":       "s390x",
		"arm":         "arm",
		"armv7":       "arm",
		"wasm32":      "wasm",
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
