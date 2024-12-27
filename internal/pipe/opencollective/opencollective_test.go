package opencollective

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "opencollective", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultTitleTemplate, ctx.Config.Announce.OpenCollective.TitleTemplate)
	require.Equal(t, defaultMessageTemplate, ctx.Config.Announce.OpenCollective.MessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `opencollective: env: environment variable "OPENCOLLECTIVE_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("skip empty slug", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: "true",
					Slug:    "", // empty
				},
			},
		}))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: "true",
					Slug:    "goreleaser",
				},
			},
		}))
		require.NoError(t, err)
		require.False(t, skip)
	})
}
