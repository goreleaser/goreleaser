package sign

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestBinarySignDescription(t *testing.T) {
	require.NotEmpty(t, BinaryPipe{}.String())
}

func TestBinarySignDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{{}},
	})

	err := BinaryPipe{}.Default(ctx)
	require.NoError(t, err)
	require.Equal(t, "gpg", ctx.Config.BinarySigns[0].Cmd)
	require.Equal(t, defaultSignatureName, ctx.Config.BinarySigns[0].Signature)
	require.Equal(t, []string{"--output", "$signature", "--detach-sig", "$artifact"}, ctx.Config.BinarySigns[0].Args)
	require.Equal(t, "binary", ctx.Config.BinarySigns[0].Artifacts)
}

func TestBinarySignDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Artifacts: "none"},
		},
	})

	err := BinaryPipe{}.Run(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestBinarySignInvalidOption(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Artifacts: "archive"},
		},
	})

	err := BinaryPipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: archive")
}

func TestBinarySkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, BinaryPipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Sign))
		require.True(t, BinaryPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			BinarySigns: []config.BinarySign{
				{},
			},
		})

		require.False(t, BinaryPipe{}.Skip(ctx))
	})
}

func TestBinaryDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		BinarySigns: []config.BinarySign{
			{Cmd: "cosign"},
			{Cmd: "gpg2"},
		},
	})

	require.Equal(t, []string{"cosign", "gpg2"}, BinaryPipe{}.Dependencies(ctx))
}
