package discourse

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnounce(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path != "/posts.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		pr := &postsRequest{}

		body, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(body, pr)
		assert.NoError(t, err)
		assert.Equal(t, "Honk v1.0.0 is out!", pr.Title)
		assert.Equal(t, "Honk v1.0.0 is out! Check it out at https://github.com/honk/honk/releases/tag/v1.0.0", pr.Raw)

		w.WriteHeader(200)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "Honk",
		Announce: config.Announce{
			Discourse: config.Discourse{
				Enabled: "true",
				Server:  ts.URL,
			},
		},
	})

	ctx.Git.CurrentTag = "v1.0.0"
	ctx.ReleaseURL = "https://github.com/honk/honk/releases/tag/v1.0.0"
	ctx.Git.URL = "https://github.com/honk/honk"

	t.Setenv("DISCOURSE_API_KEY", "XXXX-XX-XXXXX")

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discourse: config.Discourse{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discourse: config.Discourse{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `discourse: env: environment variable "DISCOURSE_API_KEY" should not be empty.`)
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`, ctx.Config.Announce.Discourse.MessageTemplate)
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
				Discourse: config.Discourse{
					Enabled: "true",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestStringer(t *testing.T) {
	require.Equal(t, "discourse", Pipe{}.String())
}

func TestLive(t *testing.T) {
	t.SkipNow()
	//t.Setenv("DISCORD_WEBHOOK_TOKEN", "TODO")

	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Discourse: config.Discourse{
				MessageTemplate: "test",
				Enabled:         "true",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}
