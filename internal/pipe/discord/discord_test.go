package discord

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "discord")
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.Discord.MessageTemplate, defaultMessageTemplate)
}

func TestAnnounceDisabled(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Announce(ctx))
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{
				Enabled:         true,
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `announce: failed to announce to discord: template: tmpl:1: unexpected "}" in operand`)
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{
				Enabled: true,
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `announce: failed to announce to discord: env: environment variable "DISCORD_WEBHOOK_ID" should not be empty`)
}

func TestAnnounceSkipAnnounce(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Discord: config.Discord{
				Enabled: true,
			},
		},
	})
	ctx.SkipAnnounce = true
	testlib.AssertSkipped(t, Pipe{}.Announce(ctx))
}
