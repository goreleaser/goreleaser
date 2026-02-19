package rust

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllowConcurrentBuilds(t *testing.T) {
	require.False(t, Default.AllowConcurrentBuilds())
}

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "cargo",
			Command: "zigbuild",
			Dir:     ".",
			Targets: defaultTargets(),
			BuildDetails: config.BuildDetails{
				Flags: []string{"--release"},
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

func TestParse(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("a-b")
		require.Error(t, err)
	})

	t.Run("triplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-apple-darwin")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-apple-darwin",
			Os:     "darwin",
			Arch:   "arm64",
			Vendor: "apple",
		}, target)
	})

	t.Run("quadruplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-pc-windows-gnullvm")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-pc-windows-gnullvm",
			Os:     "windows",
			Arch:   "arm64",
			Vendor: "pc",
			Abi:    "gnullvm",
		}, target)
	})
}

func TestIsSettingPackage(t *testing.T) {
	for name, tt := range map[string]struct {
		flags  []string
		expect bool
	}{
		"not set":   {[]string{"--release", "--something-else", "--in-the-p=middle", "--something"}, false},
		"-p":        {[]string{"--release", "-p=foo", "--something"}, true},
		"--package": {[]string{"--release", "--package=foo", "--something"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			got := isSettingPackage(tt.flags)
			require.Equal(t, tt.expect, got)
		})
	}
}
