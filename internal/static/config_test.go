package static

import (
	"bytes"
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestExampleConfig(t *testing.T) {
	cfg, err := config.LoadReader(bytes.NewReader(ExampleConfig))
	require.NoError(t, err)
	require.NotEmpty(t, ExampleConfig)
	require.Equal(t, 2, cfg.Version)
}
