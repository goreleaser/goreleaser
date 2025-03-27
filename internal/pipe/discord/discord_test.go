package discord

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
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

func TestAnnounce(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/webhooks/id/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		wm := &WebhookMessageCreate{}

		body, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(body, wm)
		assert.NoError(t, err)
		assert.Equal(t, defaultColor, strconv.Itoa(wm.Embeds[0].Color))
		assert.Equal(t, "Honk v1.0.0 is out! Check it out at https://github.com/honk/honk/releases/tag/v1.0.0", wm.Embeds[0].Description)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "Honk",
		Announce: config.Announce{
			Discord: config.Discord{
				Enabled: "true",
			},
		},
	})

	ctx.Git.CurrentTag = "v1.0.0"
	ctx.ReleaseURL = "https://github.com/honk/honk/releases/tag/v1.0.0"
	ctx.Git.URL = "https://github.com/honk/honk"

	t.Setenv("DISCORD_API", ts.URL)
	t.Setenv("DISCORD_WEBHOOK_ID", "id")
	t.Setenv("DISCORD_WEBHOOK_TOKEN", "token")

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Discord: config.Discord{
					Enabled: "true",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
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
				Enabled:         "true",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}
