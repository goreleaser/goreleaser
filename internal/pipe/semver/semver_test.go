package semver

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestValidSemver(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v1.5.2-rc1"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
	}, ctx.Semver)
}

func TestValidCalver(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("v2025.02.18-0rc1"))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      2025,
		Minor:      2,
		Patch:      18,
		Prerelease: "0rc1",
	}, ctx.Semver)
}

func TestInvalidSemver(t *testing.T) {
	ctx := testctx.New(testctx.WithCurrentTag("aaaav1.5.2-rc1"))
	err := Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "failed to parse tag 'aaaav1.5.2-rc1' as semver")
}
