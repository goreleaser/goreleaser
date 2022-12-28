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
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, Pipe{}.String(), "webhook")
}

func TestNoEndpoint(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `webhook: no endpoint url`)
}

func TestMalformedEndpoint(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL: "httxxx://example.com",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `webhook: Post "httxxx://example.com": unsupported protocol scheme "httxxx"`)
}

func TestAnnounceInvalidMessageTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     "https://example.com/webhook",
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	require.EqualError(t, Pipe{}.Announce(ctx), `webhook: template: tmpl:1: unexpected "}" in operand`)
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
		require.NoError(t, err)
		require.Equal(t, "webhook-test", string(body))

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
			},
		},
	})
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
		require.NoError(t, err)
		require.Equal(t, "webhook-test", string(body))
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		require.NoError(t, err)
	}))
	defer srv.Close()
	fmt.Println(srv.URL)
	ctx := context.New(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
				SkipTLSVerify:   true,
			},
		},
	})
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
		require.NoError(t, err)
	}))
	defer srv.Close()
	fmt.Println(srv.URL)
	ctx := context.New(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:   srv.URL,
				SkipTLSVerify: false,
			},
		},
	})
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
		require.NoError(t, err)
		require.Equal(t, "webhook-test", string(body))

		auth := r.Header.Get("Authorization")
		require.Equal(t, fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("user:pass"))), auth)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx := context.New(config.Project{
		ProjectName: "webhook-test",
		Announce: config.Announce{
			Webhook: config.Webhook{
				EndpointURL:     srv.URL,
				MessageTemplate: "{{ .ProjectName }}",
			},
		},
	})
	t.Setenv("BASIC_AUTH_HEADER_VALUE", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("user:pass"))))
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
		require.NoError(t, err)
		require.Equal(t, "webhook-test", string(body))

		customHeader := r.Header.Get("X-Custom-Header")
		require.Equal(t, "custom-value", customHeader)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(responseServer)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
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
	require.NoError(t, Pipe{}.Announce(ctx))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Announce: config.Announce{
				Webhook: config.Webhook{
					Enabled: true,
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
