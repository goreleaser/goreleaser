package semver

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/pipe"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestValidSemver(t *testing.T) {
	var ctx = context.New(config.Project{})
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
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "aaaav1.5.2-rc1"
	var err = Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse tag aaaav1.5.2-rc1 as semver")
}

func TestInvalidSemverOnSnapshots(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "aaaav1.5.2-rc1"
	ctx.Snapshot = true
	require.EqualError(t, Pipe{}.Run(ctx), pipe.ErrSnapshotEnabled.Error())
	require.Equal(t, context.Semver{
		Major:      0,
		Minor:      0,
		Patch:      0,
		Prerelease: "",
	}, ctx.Semver)
}

func TestInvalidSemverSkipValidate(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Git.CurrentTag = "aaaav1.5.2-rc1"
	ctx.SkipValidate = true
	require.EqualError(t, Pipe{}.Run(ctx), pipe.ErrSkipValidateEnabled.Error())
	require.Equal(t, context.Semver{
		Major:      0,
		Minor:      0,
		Patch:      0,
		Prerelease: "",
	}, ctx.Semver)
}
