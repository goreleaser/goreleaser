package telegram

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "telegram")
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := testctx.New()
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, ctx.Config.Announce.Telegram.MessageTemplate, defaultMessageTemplate)
		require.Equal(t, ctx.Config.Announce.Telegram.ParseMode, parseModeMarkdown)
	})
	t.Run("markdownv2 parsemode", func(t *testing.T) {
		ctx := testctx.New()
		ctx.Config.Announce.Telegram.ParseMode = parseModeMarkdown
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, ctx.Config.Announce.Telegram.ParseMode, parseModeMarkdown)
	})
	t.Run("html parsemode", func(t *testing.T) {
		ctx := testctx.New()
		ctx.Config.Announce.Telegram.ParseMode = parseModeHTML
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, ctx.Config.Announce.Telegram.ParseMode, parseModeHTML)
	})
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	t.Run("message", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Telegram: config.Telegram{
					MessageTemplate: "{{ .Foo }",
				},
			},
		})
		testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
	})
	t.Run("chatid", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Telegram: config.Telegram{
					MessageTemplate: "test",
					ChatID:          "{{ .Foo }",
				},
			},
		})
		testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
	})
	t.Run("chatid not int", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"CHAT_ID=test"},
			Announce: config.Announce{
				Telegram: config.Telegram{
					MessageTemplate: "test",
					ChatID:          "{{ .Env.CHAT_ID }}",
				},
			},
		})
		require.EqualError(t, Pipe{}.Announce(ctx), "telegram: strconv.ParseInt: parsing \"test\": invalid syntax")
	})
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Env: []string{"CHAT_ID=10"},
		Announce: config.Announce{
			Telegram: config.Telegram{
				ChatID: "{{ .Env.CHAT_ID }}",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `telegram: env: environment variable "TELEGRAM_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Telegram: config.Telegram{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestGetMessageDetails(t *testing.T) {
	t.Run("default message template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			config.Project{
				ProjectName: "foo",
				Announce: config.Announce{
					Telegram: config.Telegram{
						ChatID: "1230212",
					},
				},
			},
			testctx.WithCurrentTag("v1.0.0"),
		)
		require.NoError(t, Pipe{}.Default(ctx))
		msg, _, err := getMessageDetails(ctx)
		require.NoError(t, err)
		require.Equal(t, "foo v1\\.0\\.0 is out! Check it out at ", msg)
	})
}
