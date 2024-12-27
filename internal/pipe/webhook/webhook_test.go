package webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "webhook", Pipe{}.String())
}

func TestNoEndpoint(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `webhook: no endpoint url`)
}

func TestMalformedEndpoint(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL: "httxxx://example.com",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `webhook: Post "httxxx://example.com": unsupported protocol scheme "httxxx"`)
}

func TestAnnounceInvalidMessageTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     "https://example.com/webhook",
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

type WebHookServerMockMessage struct {
	Response string    `json:"response"`
	UUID     uuid.UUID `json:"uuid"`
}

func TestAnnounceWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "webhook-test", string(body))

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceTLSWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "webhook-test", string(body))
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	fmt.Println(srv.URL)
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
				SkipTLSVerify:   true,
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceTLSCheckCertWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	fmt.Println(srv.URL)
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:   srv.URL,
				SkipTLSVerify: false,
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Error(t, Pipe{}.Announce(ctx))
}

func TestAnnounceBasicAuthWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "webhook-test", string(body))

		auth := r.Header.Get("Authorization")
		assert.Equal(t, fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("user:pass"))), auth)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))

	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
			},
		},
	})
	t.Setenv("BASIC_AUTH_HEADER_VALUE", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("user:pass"))))
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceAdditionalHeadersWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "webhook-test", string(body))

		customHeader := r.Header.Get("X-Custom-Header")
		assert.Equal(t, "custom-value", customHeader)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceExpectedStatusCodesWebhook(t *testing.T) {
	responseServer := WebHookServerMockMessage{
		Response: "Thanks for the announcement!",
		UUID:     uuid.New(),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "webhook-test", string(body))

		w.WriteHeader(418)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:         srv.URL,
				MessageTemplate:     "{{ .ProjectName }}",
				ExpectedStatusCodes: []int{418},
			},
		},
	})
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
				Webhook: config.Webhook{
					Enabled: "true",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Webhook: config.Webhook{},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		actual := ctx.Config.Announce.Webhook
		require.NotEmpty(t, actual.MessageTemplate)
		require.NotEmpty(t, actual.ContentType)
		require.NotEmpty(t, actual.ExpectedStatusCodes)
	})
	t.Run("not empty", func(t *testing.T) {
		expected := config.Webhook{
			MessageTemplate:     "foo",
			ContentType:         "text",
			ExpectedStatusCodes: []int{200},
		}
		ctx := testctx.NewWithCfg(config.Project{
			Announce: config.Announce{
				Webhook: expected,
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		actual := ctx.Config.Announce.Webhook
		require.Equal(t, expected, actual)
	})
}
