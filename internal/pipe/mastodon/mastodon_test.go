package mastodon

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "mastodon", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultMessageTemplate, ctx.Config.Announce.Mastodon.MessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Mastodon: config.Mastodon{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Mastodon: config.Mastodon{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `mastodon: env: environment variable "MASTODON_CLIENT_ID" should not be empty; environment variable "MASTODON_CLIENT_SECRET" should not be empty; environment variable "MASTODON_ACCESS_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("skip empty server", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Mastodon: config.Mastodon{
					Enabled: "true",
					Server:  "", // empty
				},
			},
		}))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Mastodon: config.Mastodon{
					Enabled: "true",
					Server:  "https://mastodon.social",
				},
			},
		}))
		require.NoError(t, err)
		require.False(t, skip)
	})
}
