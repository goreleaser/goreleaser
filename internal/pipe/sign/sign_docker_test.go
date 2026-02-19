package sign

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDockerSignDescription(t *testing.T) {
	require.NotEmpty(t, DockerPipe{}.String())
}

func TestDockerSignDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		DockerSigns: []config.Sign{{}},
	})

	err := DockerPipe{}.Default(ctx)
	require.NoError(t, err)
	require.Equal(t, "cosign", ctx.Config.DockerSigns[0].Cmd)
	require.Empty(t, ctx.Config.DockerSigns[0].Signature)
	require.Equal(t, []string{"sign", "--key=cosign.key", "${artifact}@${digest}", "--yes"}, ctx.Config.DockerSigns[0].Args)
}

func TestDockerSignDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		DockerSigns: []config.Sign{
			{Artifacts: "none"},
		},
	})

	err := DockerPipe{}.Publish(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestDockerSignInvalidArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		DockerSigns: []config.Sign{
			{Artifacts: "foo"},
		},
	})

	err := DockerPipe{}.Publish(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestDockerSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, DockerPipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Sign))
		require.True(t, DockerPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockerSigns: []config.Sign{
				{},
			},
		})

		require.False(t, DockerPipe{}.Skip(ctx))
	})
}

func TestDockerDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		DockerSigns: []config.Sign{
			{Cmd: "cosign"},
			{Cmd: "gpg2"},
		},
	})

	require.Equal(t, []string{"cosign", "gpg2"}, DockerPipe{}.Dependencies(ctx))
}
