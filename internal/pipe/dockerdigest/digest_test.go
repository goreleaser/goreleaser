package dockerdigest

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.New()
		ctx.Config.DockerDigest.Disable = "true"
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Docker))
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})
	t.Run("normal", func(t *testing.T) {
		ctx := testctx.New()
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NotEmpty(t, ctx.Config.DockerDigest.NameTemplate)
}

func TestRun(t *testing.T) {
	ctx := testctx.New()
	ctx.Config.Dist = t.TempDir()
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:  "img1",
		Type:  artifact.DockerImage,
		Extra: artifact.Extras{artifact.ExtraDigest: "sha256:digest1"},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:  "img2",
		Type:  artifact.DockerImage, // V2,
		Extra: artifact.Extras{artifact.ExtraDigest: "sha512:digest2"},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:  "img3",
		Type:  artifact.DockerManifest,
		Extra: artifact.Extras{artifact.ExtraDigest: "digest3"},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))

	name := ctx.Config.Dist + "/digests.txt"
	require.FileExists(t, name)
	content, err := os.ReadFile(name)
	require.NoError(t, err)

	const expected = `digest1 img1
digest2 img2
digest3 img3
`
	require.Equal(t, expected, string(content))
}
