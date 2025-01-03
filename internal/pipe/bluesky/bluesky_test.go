package bluesky_test

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/pipe/bluesky"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "bluesky", bluesky.Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, bluesky.Pipe{}.Default(ctx))
	require.Equal(t, `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`, ctx.Config.Announce.Bluesky.MessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Bluesky: config.Bluesky{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, bluesky.Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Bluesky: config.Bluesky{},
		},
	})
	require.NoError(t, bluesky.Pipe{}.Default(ctx))
	require.EqualError(t, bluesky.Pipe{}.Announce(ctx), `bluesky: env: environment variable "BLUESKY_APP_PASSWORD" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := bluesky.Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					Enabled: "true",
				},
			},
		})
		skip, err := bluesky.Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestLive(t *testing.T) {
	t.SkipNow()
	t.Setenv("BLUESKY_APP_PASSWORD", "TODO")

	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Bluesky: config.Bluesky{
				MessageTemplate: "This is a sample announcement from the forthcoming {{ .ProjectName }} Bluesky support. View the details at {{ .ReleaseURL }}",
				Enabled:         "true",
				Username:        "caarlos0.dev",
			},
		},
	})

	ctx.Config.ProjectName = "Goreleaser"
	ctx.ReleaseURL = "https://goreleaser.com/customization/announce/bluesky"
	ctx.Version = "1.26.0"

	require.NoError(t, bluesky.Pipe{}.Default(ctx))
	require.NoError(t, bluesky.Pipe{}.Announce(ctx))
}
