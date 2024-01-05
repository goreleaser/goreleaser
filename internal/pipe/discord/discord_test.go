package discord

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "discord", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultMessageTemplate, ctx.Config.Announce.Discord.MessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `discord: env: environment variable "DISCORD_WEBHOOK_ID" should not be empty; environment variable "DISCORD_WEBHOOK_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Discord: config.Discord{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestLive(t *testing.T) {
	t.SkipNow()
	t.Setenv("DISCORD_WEBHOOK_ID", "TODO")
	t.Setenv("DISCORD_WEBHOOK_TOKEN", "TODO")

	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{
				MessageTemplate: "test",
				Enabled:         true,
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}
