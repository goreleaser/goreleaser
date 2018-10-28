package golang

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestAllBuildTargets(t *testing.T) {
	var build = config.Build{
		Goos: []string{
			"linux",
			"darwin",
			"freebsd",
			"openbsd",
		},
		Goarch: []string{
			"386",
			"amd64",
			"arm",
			"arm64",
		},
		Goarm: []string{
			"6",
			"7",
		},
	}
	assert.Equal(t, []string{
		"linux_386",
		"linux_amd64",
		"linux_arm_6",
		"linux_arm_7",
		"linux_arm64",
		"darwin_386",
		"darwin_amd64",
		"darwin_arm_6",
		"darwin_arm_7",
		"darwin_arm64",
		"freebsd_386",
		"freebsd_amd64",
		"freebsd_arm_6",
		"freebsd_arm_7",
		"freebsd_arm64",
		"openbsd_386",
		"openbsd_amd64",
		"openbsd_arm_6",
		"openbsd_arm_7",
		"openbsd_arm64",
	}, matrix(build))
}

func TestGoosGoarchCombos(t *testing.T) {
	var platforms = []struct {
		os    string
		arch  string
		valid bool
	}{
		// valid targets:
		{"android", "arm", true},
		{"darwin", "386", true},
		{"darwin", "amd64", true},
		{"dragonfly", "amd64", true},
		{"freebsd", "386", true},
		{"freebsd", "amd64", true},
		{"freebsd", "arm", true},
		{"linux", "386", true},
		{"linux", "amd64", true},
		{"linux", "arm", true},
		{"linux", "arm64", true},
		{"linux", "mips", true},
		{"linux", "mipsle", true},
		{"linux", "mips64", true},
		{"linux", "mips64le", true},
		{"linux", "ppc64", true},
		{"linux", "ppc64le", true},
		{"linux", "s390x", true},
		{"netbsd", "386", true},
		{"netbsd", "amd64", true},
		{"netbsd", "arm", true},
		{"openbsd", "386", true},
		{"openbsd", "amd64", true},
		{"openbsd", "arm", true},
		{"plan9", "386", true},
		{"plan9", "amd64", true},
		{"solaris", "amd64", true},
		{"windows", "386", true},
		{"windows", "amd64", true},
		// invalid targets
		{"darwin", "arm", false},
		{"darwin", "arm64", false},
		{"windows", "arm", false},
		{"windows", "arm64", false},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v valid=%v", p.os, p.arch, p.valid), func(t *testing.T) {
			assert.Equal(t, p.valid, target{p.os, p.arch, ""}.valid())
		})
	}
}
