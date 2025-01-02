package static

import (
	"bytes"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestGoExampleConfig(t *testing.T) {
	cfg, err := config.LoadReader(bytes.NewReader(GoExampleConfig))
	require.NoError(t, err)
	require.NotEmpty(t, GoExampleConfig)
	require.Equal(t, 2, cfg.Version)
}

func TestZigExampleConfig(t *testing.T) {
	cfg, err := config.LoadReader(bytes.NewReader(ZigExampleConfig))
	require.NoError(t, err)
	require.NotEmpty(t, ZigExampleConfig)
	require.Equal(t, 2, cfg.Version)
	require.Equal(t, "zig", cfg.Builds[0].Builder)
}

func TestBunExampleConfig(t *testing.T) {
	cfg, err := config.LoadReader(bytes.NewReader(BunExampleConfig))
	require.NoError(t, err)
	require.NotEmpty(t, BunExampleConfig)
	require.Equal(t, 2, cfg.Version)
	require.Equal(t, "bun", cfg.Builds[0].Builder)
}
