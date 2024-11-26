package zig

import (
	"slices"
	"strings"
	"sync"

	_ "embed"

	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
)

var (
	//go:embed all_targets.txt
	allTargetsBts []byte

	//go:embed error_targets.txt
	errTargetsBts []byte

	allTargets  []string
	errTargets  []string
	targetsOnce sync.Once
)

const keyAbi = "Abi"

// Target is a Zig build target.
type Target struct {
	// The zig formatted target (arch-os-abi).
	Target string
	Os     string
	Arch   string
	Abi    string
}

// Fields implements build.Target.
func (t Target) Fields() map[string]string {
	return map[string]string{
		tmpl.KeyOS:   t.Os,
		tmpl.KeyArch: t.Arch,
		keyAbi:       t.Abi,
	}
}

// String implements fmt.Stringer.
func (t Target) String() string {
	return t.Target
}

func convertToGoos(s string) string {
	switch s {
	case "macos":
		return "darwin"
	default:
		return s
	}
}

func convertToGoarch(s string) string {
	switch s {
	case "aarch64":
		return "arm64"
	case "x86_64":
		return "amd64"
	default:
		return s
	}
}

type targetStatus uint8

const (
	targetInvalid targetStatus = iota
	targetBroken
	targetValid
)

func (t targetStatus) String() string {
	return [3]string{
		"invalid",
		"broken",
		"valid",
	}[t]
}

func checkTarget(target string) targetStatus {
	targetsOnce.Do(func() {
		allTargets = strings.Split(string(allTargetsBts), "\n")
		errTargets = strings.Split(string(errTargetsBts), "\n")
	})

	if slices.Contains(errTargets, target) {
		return targetBroken
	}
	if slices.Contains(allTargets, target) {
		return targetValid
	}
	return targetInvalid
}

func defaultTargets() []string {
	return []string{
		"x86_64-linux",
		"x86_64-macos",
		"x86_64-windows",
		"aarch64-linux",
		"aarch64-macos",
	}
}
