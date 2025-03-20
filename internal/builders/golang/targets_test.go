package golang

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllBuildTargets(t *testing.T) {
	build := config.Build{
		Tool: "go",
		Goos: []string{
			"linux",
			"darwin",
			"freebsd",
			"openbsd",
			"windows",
			"js",
			"ios",
			"wasip1",
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
			"loong64",
			"ppc64",
			"ppc64le",
		},
		Goamd64: []string{
			"v1",
			"v2",
			"v3",
			"v4",
		},
		Go386: []string{
			"sse2",
			"softfloat",
		},
		Goarm: []string{
			"5",
			"6",
			"7",
		},
		Goarm64: []string{
			"v8.0",
			"v9.0",
		},
		Gomips: []string{
			"hardfloat",
			"softfloat",
		},
		Goppc64: []string{
			"power8",
			"power9",
			"power10",
		},
		Goriscv64: []string{
			"rva20u64",
			"rva22u64",
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
				Goarch:  "amd64",
				Goamd64: "v1",
			}, {
				Goarch: "386",
				Go386:  "sse2",
			}, {
				Goarch:  "arm64",
				Goarm64: "v8.0",
			}, {
				Goarch: "mips",
				Gomips: "hardfloat",
			}, {
				Goarch: "mipsle",
				Gomips: "hardfloat",
			}, {
				Goarch: "mips64",
				Gomips: "hardfloat",
			}, {
				Goarch: "mips64le",
				Gomips: "hardfloat",
			}, {
				Goarch:  "ppc64",
				Goppc64: "power8",
			}, {
				Goarch:  "ppc64le",
				Goppc64: "power9",
			}, {
				Goarch:    "riscv64",
				Goriscv64: "rva20u64",
			},
		},
	}

	t.Run("go 1.18", func(t *testing.T) {
		result, err := listTargets(build)
		require.NoError(t, err)
		require.Equal(t, []string{
			"linux_386_softfloat",
			"linux_amd64_v2",
			"linux_amd64_v3",
			"linux_amd64_v4",
			"linux_arm_5",
			"linux_arm_6",
			"linux_arm64_v9.0",
			"linux_mips_softfloat",
			"linux_mips64_softfloat",
			"linux_mipsle_softfloat",
			"linux_mips64le_softfloat",
			"linux_riscv64_rva22u64",
			"linux_loong64",
			"linux_ppc64_power9",
			"linux_ppc64_power10",
			"linux_ppc64le_power8",
			"linux_ppc64le_power10",
			"darwin_amd64_v2",
			"darwin_amd64_v3",
			"darwin_amd64_v4",
			"darwin_arm64_v9.0",
			"freebsd_386_softfloat",
			"freebsd_amd64_v2",
			"freebsd_amd64_v3",
			"freebsd_amd64_v4",
			"freebsd_arm_5",
			"freebsd_arm_6",
			"freebsd_arm_7",
			"freebsd_arm64_v9.0",
			"openbsd_386_softfloat",
			"openbsd_amd64_v2",
			"openbsd_amd64_v3",
			"openbsd_amd64_v4",
			"openbsd_arm64_v9.0",
			"windows_386_softfloat",
			"windows_amd64_v2",
			"windows_amd64_v3",
			"windows_amd64_v4",
			"windows_arm_5",
			"windows_arm_6",
			"windows_arm_7",
			"windows_arm64_v9.0",
			"js_wasm",
			"ios_arm64_v9.0",
			"wasip1_wasm",
		}, result)
	})

	t.Run("invalid goos", func(t *testing.T) {
		_, err := listTargets(config.Build{
			Goos:    []string{"invalid"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v2"},
		})
		require.EqualError(t, err, "invalid goos: invalid")
	})

	t.Run("invalid goarch", func(t *testing.T) {
		_, err := listTargets(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"invalid"},
		})
		require.EqualError(t, err, "invalid goarch: invalid")
	})

	t.Run("invalid goarm", func(t *testing.T) {
		_, err := listTargets(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"arm"},
			Goarm:  []string{"invalid"},
		})
		require.EqualError(t, err, "invalid goarm: invalid")
	})

	t.Run("invalid gomips", func(t *testing.T) {
		_, err := listTargets(config.Build{
			Goos:   []string{"linux"},
			Goarch: []string{"mips"},
			Gomips: []string{"invalid"},
		})
		require.EqualError(t, err, "invalid gomips: invalid")
	})

	t.Run("invalid goamd64", func(t *testing.T) {
		_, err := listTargets(config.Build{
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
		{"freebsd", "arm64", true},
		{"illumos", "amd64", true},
		{"ios", "arm64", true},
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
		{"linux", "loong64", true},
		{"netbsd", "386", true},
		{"netbsd", "amd64", true},
		{"netbsd", "arm", true},
		{"netbsd", "arm64", true},
		{"openbsd", "386", true},
		{"openbsd", "amd64", true},
		{"openbsd", "arm", true},
		{"openbsd", "arm64", true},
		{"plan9", "386", true},
		{"plan9", "amd64", true},
		{"plan9", "arm", true},
		{"solaris", "amd64", true},
		{"solaris", "sparc", true},
		{"solaris", "sparc64", true},
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
			require.Equal(t, p.valid, valid(Target{Goos: p.os, Goarch: p.arch}))
		})
	}
}

func TestListTargets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		targets, err := listTargets(config.Build{
			Goos:    []string{"linux"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v2"},
			Tool:    "go",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"linux_amd64_v2"}, targets)
	})

	t.Run("success with dir", func(t *testing.T) {
		targets, err := listTargets(config.Build{
			Goos:    []string{"linux"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v2"},
			Tool:    "go",
			Dir:     "./testdata",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"linux_amd64_v2"}, targets)
	})
}
