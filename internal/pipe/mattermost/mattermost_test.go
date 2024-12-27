package mattermost

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "mattermost", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultMessageTemplate, ctx.Config.Announce.Mattermost.MessageTemplate)
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
		skip, err := Pipe{}.Skip(testctx.New())
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Mattermost: config.Mattermost{
					Enabled: "true",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestPostWebhook(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &incomingWebhookRequest{}

		body, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(body, rc)
		assert.NoError(t, err)
		assert.Equal(t, defaultColor, rc.Attachments[0].Color)
		assert.Equal(t, "Honk v1.0.0 is out!", rc.Attachments[0].Title)
		assert.Equal(t, "Honk v1.0.0 is out! Check it out at https://github.com/honk/honk/releases/tag/v1.0.0", rc.Attachments[0].Text)

		w.WriteHeader(200)
		_, err = w.Write([]byte{})
		assert.NoError(t, err)
	}))
	defer ts.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "Honk",
		Announce: config.Announce{
			Mattermost: config.Mattermost{
				Enabled: "true",
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
