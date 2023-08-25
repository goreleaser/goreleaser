package mattermost

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "mattermost")
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ctx.Config.Announce.Mattermost.MessageTemplate, defaultMessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Mattermost: config.Mattermost{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Mattermost: config.Mattermost{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `mattermost: env: environment variable "MATTERMOST_WEBHOOK" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Mattermost: config.Mattermost{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPostWebhook(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &incomingWebhookRequest{}

		body, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(body, rc)
		require.NoError(t, err)
		require.Equal(t, defaultColor, rc.Attachments[0].Color)
		require.Equal(t, "Honk v1.0.0 is out!", rc.Attachments[0].Title)
		require.Equal(t, "Honk v1.0.0 is out! Check it out at https://github.com/honk/honk/releases/tag/v1.0.0", rc.Attachments[0].Text)

		w.WriteHeader(200)
		_, err = w.Write([]byte{})
		require.NoError(t, err)
	}))
	defer ts.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "Honk",
		Announce: config.Announce{
			Mattermost: config.Mattermost{
				Enabled: true,
			},
		},
	})

	ctx.Git.CurrentTag = "v1.0.0"
	ctx.ReleaseURL = "https://github.com/honk/honk/releases/tag/v1.0.0"
	ctx.Git.URL = "https://github.com/honk/honk"

	t.Setenv("MATTERMOST_WEBHOOK", ts.URL)

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}
