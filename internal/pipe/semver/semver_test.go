package semver

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestValidSemver(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Git.CurrentTag = "v1.5.2-rc1"
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, context.Semver{
		Major:      1,
		Minor:      5,
		Patch:      2,
		Prerelease: "rc1",
	}, ctx.Semver)
}

func TestInvalidSemver(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Git.CurrentTag = "aaaav1.5.2-rc1"
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse tag 'aaaav1.5.2-rc1' as semver")
}
