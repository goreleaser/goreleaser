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
			"linux-386-softfloat",
			"linux-amd64-v2",
			"linux-amd64-v3",
			"linux-amd64-v4",
			"linux-arm-5",
			"linux-arm-6",
			"linux-arm64-v9.0",
			"linux-mips-softfloat",
			"linux-mips64-softfloat",
			"linux-mipsle-softfloat",
			"linux-mips64le-softfloat",
			"linux-riscv64-rva22u64",
			"linux-loong64",
			"linux-ppc64-power9",
			"linux-ppc64-power10",
			"linux-ppc64le-power8",
			"linux-ppc64le-power10",
			"darwin-amd64-v2",
			"darwin-amd64-v3",
			"darwin-amd64-v4",
			"darwin-arm64-v9.0",
			"freebsd-386-softfloat",
			"freebsd-amd64-v2",
			"freebsd-amd64-v3",
			"freebsd-amd64-v4",
			"freebsd-arm-5",
			"freebsd-arm-6",
			"freebsd-arm-7",
			"freebsd-arm64-v9.0",
			"openbsd-386-softfloat",
			"openbsd-amd64-v2",
			"openbsd-amd64-v3",
			"openbsd-amd64-v4",
			"openbsd-arm64-v9.0",
			"openbsd-riscv64-rva22u64",
			"windows-386-softfloat",
			"windows-amd64-v2",
			"windows-amd64-v3",
			"windows-amd64-v4",
			"windows-arm-5",
			"windows-arm-6",
			"windows-arm-7",
			"windows-arm64-v9.0",
			"js-wasm",
			"ios-arm64-v9.0",
			"wasip1-wasm",
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
		// experimental targets:
		{os: "openbsd", arch: "riscv64", valid: true},
		// broken/to-be-removed:
		{os: "windows", arch: "arm", valid: true},
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
		require.Equal(t, []string{"linux-amd64-v2"}, targets)
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
		require.Equal(t, []string{"linux-amd64-v2"}, targets)
	})
}
