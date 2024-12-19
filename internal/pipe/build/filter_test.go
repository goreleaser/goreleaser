package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

var filterTestTargets = []string{
	"linux_amd64_v1",
	"linux_arm64",
	"linux_riscv64",
	"darwin_amd64_v1",
	"darwin_amd64_v2",
	"darwin_arm64",
	"darwin_arm_7",
}

func TestFilter(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		ctx := testctx.New()
		require.Equal(t, filterTestTargets, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("target", func(t *testing.T) {
		ctx := testctx.New(func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "darwin_amd64_v1"
		})
		require.Equal(t, []string{
			"darwin_amd64_v1",
		}, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("incomplete target", func(t *testing.T) {
		ctx := testctx.New(func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "darwin_amd64"
		})
		require.Equal(t, []string{
			"darwin_amd64_v1",
		}, filter(ctx, config.Build{
			Builder: "go",
			Targets: filterTestTargets,
		}))
	})

	t.Run("target no match", func(t *testing.T) {
		ctx := testctx.New(func(ctx *context.Context) {
			ctx.Partial = true
			ctx.PartialTarget = "linux_amd64_v1"
		})
		require.Empty(t, filter(ctx, config.Build{
			Builder: "go",
			Targets: []string{"darwin_amd64_v1"},
		}))
	})
}
