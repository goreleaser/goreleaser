package semver

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestValidSemver(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v1.5.2-rc1+ng"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
		Metadata:   "ng",
	}, ctx.Semver)
	require.Equal(t, "1.5.2-rc1+ng", ctx.Version)
}

func TestValidSemverWithoutV(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("1.5.2-rc1+ng"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
		Metadata:   "ng",
	}, ctx.Semver)
	require.Equal(t, "1.5.2-rc1+ng", ctx.Version)
}

func TestTagWithPrefix(t *testing.T) {
	ctx := testctx.New(
		testctx.WithCurrentTag("component_v1.5.2-rc1+ng"),
		testctx.WithTagPrefixes([]string{"component_"}),
	)
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
		Metadata:   "ng",
	}, ctx.Semver)
	require.Equal(t, "1.5.2-rc1+ng", ctx.Version)
}

func TestTagWithPrefixTemplated(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{ProjectName: "component"},
		testctx.WithCurrentTag("component_v1.5.2-rc1+ng"),
		testctx.WithTagPrefixes([]string{"{{ .ProjectName }}_"}),
	)
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
		Metadata:   "ng",
	}, ctx.Semver)
	require.Equal(t, "1.5.2-rc1+ng", ctx.Version)
}

func TestTagWithMultiplePrefixes(t *testing.T) {
	ctx := testctx.New(
		testctx.WithCurrentTag("component_v1.5.2-rc1+ng"),
		testctx.WithTagPrefixes([]string{"component_", "app_"}),
	)
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
		Metadata:   "ng",
	}, ctx.Semver)
	require.Equal(t, "1.5.2-rc1+ng", ctx.Version)
}

func TestTagWithWrongPrefixes(t *testing.T) {
	ctx := testctx.New(
		testctx.WithCurrentTag("component_v1.5.2-rc1+ng"),
		testctx.WithTagPrefixes([]string{"not_matching_prefix"}),
	)
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tag 'component_v1.5.2-rc1+ng' has none of expected prefixes [not_matching_prefix]")
}

func TestTagWithoutPrefix(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("aaaav1.5.2-rc1+ng"))
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse tag 'aaaav1.5.2-rc1+ng' as semver")
}

func TestTagPrefixNotMatches(t *testing.T) {
	ctx := testctx.New(
		testctx.WithCurrentTag("v1.5.2-rc1+ng"),
		testctx.WithTagPrefixes([]string{"component_"}),
	)
	err := Pipe{}.Run(ctx)
	require.Contains(t, err.Error(), "tag 'v1.5.2-rc1+ng' has none of expected prefixes [component_]")
}
