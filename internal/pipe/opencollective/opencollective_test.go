package opencollective

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "opencollective")
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.OpenCollective.TitleTemplate, defaultTitleTemplate)
	require.Equal(t, ctx.Config.Announce.OpenCollective.MessageTemplate, defaultMessageTemplate)
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
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("skip empty slug", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: true,
					Slug:    "", // empty
				},
			},
		})))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: true,
					Slug:    "goreleaser",
				},
			},
		})))
	})
}
