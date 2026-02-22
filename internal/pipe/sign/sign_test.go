package sign

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSignDefault(t *testing.T) {
	_ = testlib.Mktmp(t)
	testlib.GitInit(t)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Signs: []config.Sign{{}},
	})

	setGpg(t, ctx, "") // force empty gpg.program

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "gpg", ctx.Config.Signs[0].Cmd)
	require.Equal(t, "${artifact}.sig", ctx.Config.Signs[0].Signature)
	require.Equal(t, []string{"--output", "$signature", "--detach-sig", "$artifact"}, ctx.Config.Signs[0].Args)
	require.Equal(t, "none", ctx.Config.Signs[0].Artifacts)
}

func TestDefaultGpgFromGitConfig(t *testing.T) {
	_ = testlib.Mktmp(t)
	testlib.GitInit(t)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Signs: []config.Sign{{}},
	})

	setGpg(t, ctx, "not-really-gpg")

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "not-really-gpg", ctx.Config.Signs[0].Cmd)
}

func TestSignDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Signs: []config.Sign{{Artifacts: "none"}}})
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestSignInvalidArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Signs: []config.Sign{{Artifacts: "foo"}}})
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestSeveralSignsWithTheSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Signs: []config.Sign{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 signs with the ID 'a', please fix your config")
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Sign))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Signs: []config.Sign{
				{},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Signs: []config.Sign{
			{Cmd: "cosign"},
			{Cmd: "gpg2"},
		},
	})

	require.Equal(t, []string{"cosign", "gpg2"}, Pipe{}.Dependencies(ctx))
}

func setGpg(tb testing.TB, ctx *context.Context, p string) {
	tb.Helper()
	_, err := git.Run(ctx, "config", "--local", "--add", "gpg.program", p)
	require.NoError(tb, err)
}
