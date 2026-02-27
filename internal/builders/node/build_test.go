package node

import (
	"testing"
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
		"linux-x64": {
			Target: "linux-x64",
			Os:     "linux",
			Arch:   "x64",
		},
		"darwin-arm64": {
			Target: "darwin-arm64",
			Os:     "darwin",
			Arch:   "arm64",
		},
		"linux-arm64": {
			Target: "linux-arm64",
			Os:     "linux",
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

func TestIsValid(t *testing.T) {
	for _, target := range []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"win-x64",
		"win-arm64",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("bun-foo-bar"))
	})
}
