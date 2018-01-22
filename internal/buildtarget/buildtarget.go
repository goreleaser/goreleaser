package buildtarget

import (
	"fmt"
	"runtime"
)

// Runtime is the current runtime build target
var Runtime = Target{runtime.GOOS, runtime.GOARCH, ""}

// New build Target
func New(goos, goarch, goarm string) Target {
	return Target{goos, goarch, goarm}
}

// Target is a build target
type Target struct {
	OS, Arch, Arm string
}

// Env returns the current Target as environment variables
func (t Target) Env() []string {
	return []string{
		"GOOS=" + t.OS,
		"GOARCH=" + t.Arch,
		"GOARM=" + t.Arm,
	}
}

func (t Target) String() string {
	// TODO: maybe replace this as suggested to OS_ArchArm?
	return fmt.Sprintf("%v%v%v", t.OS, t.Arch, t.Arm)
}

// PrettyString is a prettier version of the String method.
func (t Target) PrettyString() string {
	return fmt.Sprintf("%v/%v%v", t.OS, t.Arch, t.Arm)
}
