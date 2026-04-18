package node

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"node"}, Default.Dependencies())
}

func TestAllowConcurrentBuilds(t *testing.T) {
	require.False(t, Default.AllowConcurrentBuilds())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"linux-x64":    {Target: "linux-x64", Os: "linux", Arch: "x64"},
		"darwin-arm64": {Target: "darwin-arm64", Os: "darwin", Arch: "arm64"},
		"win-arm64":    {Target: "win-arm64", Os: "win", Arch: "arm64"},
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
	t.Run("unknown target", func(t *testing.T) {
		_, err := Default.Parse("plan9-amd64")
		require.Error(t, err)
	})
}

func TestIsValid(t *testing.T) {
	for _, target := range []string{
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"win-arm64",
		"win-x64",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
	}
	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("plan9-amd64"))
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, ".", build.Dir)
		require.Equal(t, "index.js", build.Main)
		require.ElementsMatch(t, defaultTargets(), build.Targets)
	})

	t.Run("respects main override", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{Main: "src/cli.mjs"})
		require.NoError(t, err)
		require.Equal(t, "src/cli.mjs", build.Main)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"plan9-amd64"},
		})
		require.Error(t, err)
	})

	t.Run("rejects tool", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{Tool: "node"})
		require.ErrorContains(t, err, "tool is not supported")
	})

	t.Run("rejects command", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{Command: "build"})
		require.ErrorContains(t, err, "command is not supported")
	})

	t.Run("rejects flags", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			BuildDetails: config.BuildDetails{Flags: []string{"--x"}},
		})
		require.ErrorContains(t, err, "flags is not supported")
	})
}

func TestCurrentTarget(t *testing.T) {
	got := CurrentTarget()
	require.NotEmpty(t, got)
	// Must always parse cleanly under our own rules.
	require.True(t, isValid(got), "%s should be a valid target", got)
}

func TestConvertHelpers(t *testing.T) {
	require.Equal(t, "amd64", convertToGoarch("x64"))
	require.Equal(t, "arm64", convertToGoarch("arm64"))
	require.Equal(t, "windows", convertToGoos("win"))
	require.Equal(t, "linux", convertToGoos("linux"))
}
