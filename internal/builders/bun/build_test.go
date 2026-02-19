package bun

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestAllowConcurrentBuilds(t *testing.T) {
	require.False(t, Default.AllowConcurrentBuilds())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"linux-x64-modern": {
			Target: "bun-linux-x64-modern",
			Os:     "linux",
			Arch:   "x64",
			Type:   "modern",
		},
		"darwin-arm64": {
			Target: "bun-darwin-arm64",
			Os:     "darwin",
			Arch:   "arm64",
		},
		"bun-linux-arm64": {
			Target: "bun-linux-arm64",
			Os:     "linux",
			Arch:   "arm64",
		},
		"bun-linux-arm64-musl": {
			Target: "bun-linux-arm64-musl",
			Os:     "linux",
			Arch:   "arm64",
			Abi:    "musl",
		},
		"linux-arm64-musl": {
			Target: "bun-linux-arm64-musl",
			Os:     "linux",
			Arch:   "arm64",
			Abi:    "musl",
		},
		"bun-linux-x64-musl-modern": {
			Target: "bun-linux-x64-musl-modern",
			Os:     "linux",
			Arch:   "x64",
			Abi:    "musl",
			Type:   "modern",
		},
		"linux-x64-musl-modern": {
			Target: "bun-linux-x64-musl-modern",
			Os:     "linux",
			Arch:   "x64",
			Abi:    "musl",
			Type:   "modern",
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
			Tool:    "bun",
			Command: "build",
			Dir:     ".",
			Main:    ".",
			Targets: defaultTargets(),
			BuildDetails: config.BuildDetails{
				Flags: []string{"--compile"},
			},
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})
}

func TestIsValid(t *testing.T) {
	for _, target := range []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-arm64-musl",
		"linux-x64-modern",
		"linux-x64-musl",
		"linux-x64-musl-baseline",
		"linux-x64-musl-modern",
		"windows-x64-modern",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid("bun-"+target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("bun-foo-bar"))
	})
}
