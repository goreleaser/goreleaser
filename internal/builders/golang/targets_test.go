package golang

import (
	"fmt"
	"os"
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
		"linux_riscv64",
		"darwin_amd64",
		"darwin_arm64",
		"freebsd_386",
		"freebsd_amd64",
		"freebsd_arm_6",
		"freebsd_arm_7",
		"freebsd_arm64",
		"openbsd_386",
		"openbsd_amd64",
		"openbsd_arm64",
		"windows_386",
		"windows_amd64",
		"windows_arm_6",
		"windows_arm_7",
		"js_wasm",
	}, result)
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
		{"js", "wasm", true},
		// invalid targets
		{"darwin", "386", false},
		{"darwin", "arm", false},
		{"windows", "arm64", false},
		{"windows", "riscv64", false},
	}
	for _, p := range platforms {
		t.Run(fmt.Sprintf("%v %v valid=%v", p.os, p.arch, p.valid), func(t *testing.T) {
			require.Equal(t, p.valid, valid(target{p.os, p.arch, "", ""}))
		})
	}
}

func Test_isGo116orLater(t *testing.T) {
	tests := []struct {
		name    string
		build   config.Build
		version string
		want    bool
		wantErr bool
	}{
		{
			name:    "less than go1.16",
			version: "go version go1.15.13 linux/amd64",
			want:    false,
		},
		{
			name:    "go1.16",
			version: "go version go1.16 linux/amd64",
			want:    true,
		},
		{
			name:    "patch to go1.16",
			version: "go version go1.16.5 linux/amd64",
			want:    true,
		},
		{
			name:    "greater than go1.16",
			version: "go version go1.17 linux/amd64",
			want:    true,
		},
		{
			name:    "beta greater than go1.16",
			version: "go version go1.17beta1 linux/amd64",
			want:    true,
		},
		{
			name:    "rc greater than go1.16",
			version: "go version go1.17rc1 linux/amd64",
			want:    true,
		},
		{
			name:    "go2",
			version: "go version go2 linux/amd64",
			want:    true,
		},
		{
			name:    "invalid version",
			version: "go not a version",
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatalf("unable to setup temp file for isGo116orLater: %v", err)
			}
			t.Cleanup(func() { os.Remove(f.Name()) })
			fmt.Fprintf(f, "#!/bin/sh\necho %q\n", tt.version)
			if err := f.Chmod(0o700); err != nil {
				t.Fatalf("unable to setup temp file for isGo116orLater: %v", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("unable to setup temp file for isGo116orLater: %v", err)
			}
			tt.build.GoBinary = f.Name()
			got, err := isGo116orLater(tt.build)
			if (err != nil) != tt.wantErr {
				t.Errorf("isGo116orLater() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isGo116orLater() = %v, want %v", got, tt.want)
			}
		})
	}
}
