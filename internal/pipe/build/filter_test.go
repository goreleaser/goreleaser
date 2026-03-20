package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var filterTestTargets = []string{
	"linux-amd64-v1",
	"linux-arm64",
	"linux-riscv64",
	"darwin-amd64-v1",
	"darwin-amd64-v2",
	"darwin-arm64",
	"darwin-arm-7",
}

func TestFilter(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.Equal(t, filterTestTargets, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("target", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "darwin-amd64-v1"
		})

		require.Equal(t, []string{
			"darwin-amd64-v1",
		}, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("incomplete target", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "darwin_amd64"
		})

		require.Equal(t, []string{
			"darwin-amd64-v1",
		}, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("target no match", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "linux_amd64_v1"
		})

		require.Empty(t, filter(ctx, config.Build{
			Builder: "go",
			Targets: []string{"darwin-amd64-v1"},
		}))
	})
}
