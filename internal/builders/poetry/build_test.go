package poetry

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := Default.Parse(defaultTarget)
		require.NoError(t, err)
		require.IsType(t, Target{}, got)
	})
	t.Run("invalid", func(t *testing.T) {
		got, err := Default.Parse(defaultTarget)
		require.NoError(t, err)
		require.IsType(t, Target{}, got)
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{
			Dir: "./testdata",
			InternalDefaults: config.BuildInternalDefaults{
				Binary: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "poetry",
			Command: "build",
			Dir:     "./testdata",
			Targets: []string{defaultTarget},
			InternalDefaults: config.BuildInternalDefaults{
				Binary: true,
			},
		}, build)
	})

	t.Run("user set binary", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:    "./testdata",
			Binary: "a",
		})
		require.ErrorIs(t, err, errSetBinary)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:     "./testdata",
			Targets: []string{"a-b"},
		})
		require.ErrorIs(t, err, errTargets)
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:  "./testdata",
			Main: "something",
		})
		require.Error(t, err)
	})
}
