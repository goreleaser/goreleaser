package healthcheck

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	ctx := testctx.New()
	require.Equal(t, []string{"git", "go"}, system{}.Dependencies(ctx))
}

func TestStringer(t *testing.T) {
	require.NotEmpty(t, system{}.String())
}
