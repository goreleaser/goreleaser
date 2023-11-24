package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var filterTestTargets = []string{
	"linux_amd64_v1",
	"linux_arm64",
	"linux_riscv64",
	"darwin_amd64_v1",
	"darwin_amd64_v2",
	"darwin_arm64",
}

func TestFilter(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		ctx := testctx.New()
		require.Equal(t, filterTestTargets, filter(ctx, filterTestTargets))
	})

	t.Run("target", func(t *testing.T) {
		ctx := testctx.New(func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "darwin_amd64"
		})
		require.Equal(t, []string{
			"darwin_amd64_v1",
		}, filter(ctx, filterTestTargets))
	})

	t.Run("target no match", func(t *testing.T) {
		ctx := testctx.New(func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "linux_amd64_v1"
		})
		require.Empty(t, filter(ctx, []string{"darwin_amd64_v1"}))
	})
}
