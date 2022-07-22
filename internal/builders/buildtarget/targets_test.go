package buildtarget

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllBuildTargets(t *testing.T) {
	build := config.Build{
		GoBinary: "go",
		Goos: []string{
			"linux",
			"darwin",
			"freebsd",
			"openbsd",
			"windows",
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
			"riscv64",
		},
		Goarm: []string{
			"6",
			"7",
		},
		Gomips: []string{
			"hardfloat",
			"softfloat",
		},
		Goamd64: []string{
			"v1",
			"v2",
			"v3",
			"v4",
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
			}, {
				Goarch:  "amd64",
				Goamd64: "v3",
			},
		},
	}

	t.Run("go 1.18", func(t *testing.T) {
		result, err := List(build)
		require.NoError(t, err)
		require.Equal(t, []string{
			"linux_386",
			"linux_amd64_v1",
			"linux_amd64_v2",
			"linux_amd64_v4",
			"linux_arm_6",
			"linux_arm64",
			"linux_mips_hardfloat",
			"linux_mips_softfloat",
			"linux_mips64_softfloat",
			"linux_mipsle_hardfloat",
			"linux_mipsle_softfloat",
			"linux_mips64le_hardfloat",
			"linux_riscv64",
			"darwin_amd64_v1",
			"darwin_amd64_v2",
			"darwin_amd64_v4",
			"darwin_arm64",
			"freebsd_386",
			"freebsd_amd64_v1",
			"freebsd_amd64_v2",
			"freebsd_amd64_v4",
			"freebsd_arm_6",
			"freebsd_arm_7",
			"freebsd_arm64",
			"openbsd_386",
			"openbsd_amd64_v1",
			"openbsd_amd64_v2",
			"openbsd_amd64_v4",
			"openbsd_arm64",
			"windows_386",
			"windows_amd64_v1",
			"windows_amd64_v2",
			"windows_amd64_v4",
			"windows_arm_6",
			"windows_arm_7",
			"windows_arm64",
			"js_wasm",
		}, result)
	})

	t.Run("invalid goos", func(t *testing.T) {
		_, err := List(config.Build{
			Goos:    []string{"invalid"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v2"},
		})
		require.EqualError(t, err, "invalid goos: invalid")
	})

	t.Run("invalid goarch", func(t *testing.T) {
		_, err := List(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"invalid"},
		})
		require.EqualError(t, err, "invalid goarch: invalid")
	})

	t.Run("invalid goarm", func(t *testing.T) {
		_, err := List(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"arm"},
			Goarm:  []string{"invalid"},
		})
		require.EqualError(t, err, "invalid goarm: invalid")
	})

	t.Run("invalid gomips", func(t *testing.T) {
		_, err := List(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"mips"},
			Gomips: []string{"invalid"},
		})
		require.EqualError(t, err, "invalid gomips: invalid")
	})

	t.Run("invalid goamd64", func(t *testing.T) {
		_, err := List(config.Build{
			Goos:    []string{"linux"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"invalid"},
		})
		require.EqualError(t, err, "invalid goamd64: invalid")
	})
}

func TestGoosGoarchCombos(t *testing.T) {
	platforms := []struct {
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
		{"linux", "riscv64", true},
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
		{"windows", "arm", true},
		{"windows", "arm64", true},
		{"js", "wasm", true},
		// invalid targets
		{"darwin", "386", false},
		{"darwin", "arm", false},
		{"windows", "riscv64", false},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v valid=%v", p.os, p.arch, p.valid), func(t *testing.T) {
			require.Equal(t, p.valid, valid(target{p.os, p.arch, "", "", ""}))
		})
	}
}

func TestList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		targets, err := List(config.Build{
			Goos:     []string{"linux"},
			Goarch:   []string{"amd64"},
			Goamd64:  []string{"v2"},
			GoBinary: "go",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"linux_amd64_v2"}, targets)
	})

	t.Run("success with dir", func(t *testing.T) {
		targets, err := List(config.Build{
			Goos:     []string{"linux"},
			Goarch:   []string{"amd64"},
			Goamd64:  []string{"v2"},
			GoBinary: "go",
			Dir:      "./testdata",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"linux_amd64_v2"}, targets)
	})
}
