package skips_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	for expect, keys := range map[string][]skips.Key{
		"":                     nil,
		"ko and sbom":          {skips.SBOM, skips.Ko},
		"before, ko, and sbom": {skips.SBOM, skips.Ko, skips.Before},
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
	require.ElementsMatch(t, []string{"publish", "announce"}, slices.Collect(maps.Keys(ctx.Skips)))
}

func TestSetAllowed(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := testctx.New()
		require.NoError(t, skips.SetBuild(ctx, "", "validate", ""))
		require.ElementsMatch(t, []string{"validate"}, slices.Collect(maps.Keys(ctx.Skips)))
	})
	t.Run("error", func(t *testing.T) {
		ctx := testctx.New()
		require.ErrorContains(t, skips.SetBuild(ctx, "validate", "", "publish", ""), "--skip=publish is not allowed.")
		require.ElementsMatch(t, []string{"validate"}, slices.Collect(maps.Keys(ctx.Skips)))
	})
}

func TestComplete(t *testing.T) {
	require.Equal(
		t,
		[]string{"announce", "archive", "aur", "aur-source"},
		skips.Release.Complete("a"),
	)
}
