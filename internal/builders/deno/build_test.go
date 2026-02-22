package deno

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
		"x86_64-pc-windows-msvc": {
			Target: "x86_64-pc-windows-msvc",
			Os:     "windows",
			Arch:   "x86_64",
			Abi:    "msvc",
			Vendor: "pc",
		},
		"aarch64-apple-darwin": {
			Target: "aarch64-apple-darwin",
			Os:     "darwin",
			Arch:   "aarch64",
			Vendor: "apple",
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
			Tool:    "deno",
			Command: "compile",
			Dir:     ".",
			Main:    "main.ts",
			Targets: defaultTargets(),
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
		"x86_64-pc-windows-msvc",
		"x86_64-apple-darwin",
		"aarch64-apple-darwin",
		"x86_64-unknown-linux-gnu",
		"aarch64-unknown-linux-gnu",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("foo-bar"))
	})
}
