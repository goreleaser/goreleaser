//go:build integration

package cmd

import (
	"bytes"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/internal/static"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestIntegrationInitExampleConfigsAreNotDeprecated(t *testing.T) {
	checkExample(t, static.GoExampleConfig)
	checkExample(t, static.ZigExampleConfig)
	checkExample(t, static.BunExampleConfig)
	checkExample(t, static.DenoExampleConfig)
	checkExample(t, static.RustExampleConfig)
}

func checkExample(t *testing.T, exampleConfig []byte) {
	t.Helper()
	cfg, err := config.LoadReader(bytes.NewReader(exampleConfig))
	require.NoError(t, err)
	ctx := testctx.WrapWithCfg(t.Context(), cfg)
	err = defaults.Pipe{}.Run(ctx)
	require.NoError(t, err)
	require.False(t, ctx.Deprecated)
}
