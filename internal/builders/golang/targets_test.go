package golang

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllBuildTargets(t *testing.T) {
	var build = config.Build{
		Goos: []string{
			"linux",
			"darwin",
			"freebsd",
			"openbsd",
			"js",
		},
		Goarch: []string{
			"386",
			"amd64",
			"arm",
			"arm64",
			"wasm",
			"mips",
			"mips64",
			"mipsle",
			"mips64le",
		},
		Goarm: []string{
			"6",
			"7",
		},
		Gomips: []string{
			"hardfloat",
			"softfloat",
		},
		Ignore: []config.IgnoredBuild{
			{
				Goos:   "linux",
				Goarch: "arm",
				Goarm:  "7",
			}, {
				Goos:   "openbsd",
				Goarch: "arm",
			}, {
				Goarch: "mips64",
				Gomips: "hardfloat",
			}, {
				Goarch: "mips64le",
				Gomips: "softfloat",
			},
		},
	}
	result, err := matrix(build)
	require.NoError(t, err)
	require.Equal(t, []string{
		"linux_386",
		"linux_amd64",
		"linux_arm_6",
		"linux_arm64",
		"linux_mips_hardfloat",
		"linux_mips_softfloat",
		"linux_mips64_softfloat",
		"linux_mipsle_hardfloat",
		"linux_mipsle_softfloat",
		"linux_mips64le_hardfloat",
		"darwin_amd64",
		"freebsd_386",
		"freebsd_amd64",
		"freebsd_arm_6",
		"freebsd_arm_7",
		"freebsd_arm64",
		"openbsd_386",
		"openbsd_amd64",
		"openbsd_arm64",
		"js_wasm",
	}, result)
}

func TestGoosGoarchCombos(t *testing.T) {
	var platforms = []struct {
		os    string
		arch  string
		valid bool
	}{
		// valid targets:
		{"aix", "ppc64", true},
		{"android", "386", true},
		{"android", "amd64", true},
		{"android", "arm", true},
		{"android", "arm64", true},
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"dragonfly", "amd64", true},
		{"freebsd", "386", true},
		{"freebsd", "amd64", true},
		{"freebsd", "arm", true},
		{"illumos", "amd64", true},
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
		{"plan9", "arm", true},
		{"solaris", "amd64", true},
		{"windows", "386", true},
		{"windows", "amd64", true},
		{"js", "wasm", true},
		// invalid targets
		{"darwin", "386", false},
		{"darwin", "arm", false},
		{"windows", "arm", false},
		{"windows", "arm64", false},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v valid=%v", p.os, p.arch, p.valid), func(t *testing.T) {
			require.Equal(t, p.valid, valid(target{p.os, p.arch, "", ""}))
		})
	}
}
