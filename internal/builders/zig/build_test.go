package zig

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"x86_64-linux": {
			Target: "x86_64-linux",
			Os:     "linux",
			Arch:   "amd64",
		},
		"x86_64-linux-gnu": {
			Target: "x86_64-linux-gnu",
			Os:     "linux",
			Arch:   "amd64",
			Abi:    "gnu",
		},
		"aarch64-linux-gnu": {
			Target: "aarch64-linux-gnu",
			Os:     "linux",
			Arch:   "arm64",
			Abi:    "gnu",
		},
		"aarch64-linux": {
			Target: "aarch64-linux",
			Os:     "linux",
			Arch:   "arm64",
		},
		"aarch64-macos": {
			Target: "aarch64-macos",
			Os:     "darwin",
			Arch:   "arm64",
		},
	} {
		t.Run(target, func(t *testing.T) {
			got, err := Default.Parse(target)
			require.NoError(t, err)
			require.IsType(t, Target{}, got)
			require.Equal(t, dst, got.(Target))
		})
	}
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("linux")
		require.Error(t, err)
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "zig",
			Command: "build",
			Dir:     ".",
			Targets: defaultTargets(),
			BuildDetails: config.BuildDetails{
				Flags: []string{"-Doptimize=ReleaseSafe"},
			},
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Main: "something",
		})
		require.Error(t, err)
	})
}
