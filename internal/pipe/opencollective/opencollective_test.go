package opencollective

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "opencollective")
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.OpenCollective.TitleTemplate, defaultTitleTemplate)
	require.Equal(t, ctx.Config.Announce.OpenCollective.MessageTemplate, defaultMessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `opencollective: template: tmpl:1: unexpected "}" in operand`)
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `opencollective: env: environment variable "OPENCOLLECTIVE_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("skip empty slug", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: true,
					Slug:    "", // empty
				},
			},
		})))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(context.New(config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: true,
					Slug:    "goreleaser",
				},
			},
		})))
	})
}
