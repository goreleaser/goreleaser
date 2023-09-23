package skips_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestString(t *testing.T) {
	for expect, keys := range map[string][]skips.Key{
		"":                    nil,
		"ko and sbom":         {skips.SBOM, skips.Ko},
		"before, ko and sbom": {skips.SBOM, skips.Ko, skips.Before},
	} {
		t.Run(expect, func(t *testing.T) {
			ctx := testctx.New(testctx.Skip(keys...))
			require.Equal(t, expect, skips.String(ctx))
		})
	}
}

func TestAny(t *testing.T) {
	t.Run("false", func(t *testing.T) {
		ctx := testctx.New()
		require.False(t, skips.Any(ctx, skips.Release...))
	})
	t.Run("true", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Publish))
		require.True(t, skips.Any(ctx, skips.Release...))
	})
}

func TestSet(t *testing.T) {
	ctx := testctx.New()
	skips.Set(ctx, skips.Publish, skips.Announce)
	require.Equal(t, []string{"publish", "announce"}, maps.Keys(ctx.Skips))
}
