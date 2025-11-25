package bluesky

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	p := New()
	require.Equal(t, defaultPDSURL, p.pdsURL)
}

func TestStringer(t *testing.T) {
	require.Equal(t, "bluesky", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	t.Run("default template", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`, ctx.Config.Announce.Bluesky.MessageTemplate)
	})

	t.Run("custom template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					MessageTemplate: "custom template",
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, "custom template", ctx.Config.Announce.Bluesky.MessageTemplate)
	})
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			Bluesky: config.Bluesky{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			Bluesky: config.Bluesky{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `bluesky: env: environment variable "BLUESKY_APP_PASSWORD" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.Wrap(t.Context()))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					Enabled: "true",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.False(t, skip)
	})

	t.Run("skip with false", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					Enabled: "false",
				},
			},
		})
		skip, err := Pipe{}.Skip(ctx)
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					Enabled: "{{ .Invalid }",
				},
			},
		})
		_, err := Pipe{}.Skip(ctx)
		require.Error(t, err)
	})
}

func TestAnnounceWithMockServer(t *testing.T) {
	t.Run("success with release URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"accessJwt":  "test-access-token",
					"refreshJwt": "test-refresh-token",
					"handle":     "testuser.bsky.social",
					"did":        "did:plc:test123",
				}))
			case "/xrpc/com.atproto.repo.createRecord":
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"uri": "at://did:plc:test123/app.bsky.feed.post/test",
					"cid": "testcid",
				}))
			default:
				assert.Failf(t, "unexpected request", "unexpected request to %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(server.Close)

		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "test-project",
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					MessageTemplate: "Release {{ .ProjectName }} {{ .Tag }} at {{ .ReleaseURL }}",
					Username:        "testuser.bsky.social",
				},
			},
		})
		ctx.Version = "v1.0.0"
		ctx.ReleaseURL = "https://github.com/test/test/releases/tag/v1.0.0"

		require.NoError(t, Pipe{}.Default(ctx))
		t.Setenv("BLUESKY_APP_PASSWORD", "test-password")

		pipe := Pipe{pdsURL: server.URL}
		require.NoError(t, pipe.Announce(ctx))
	})

	t.Run("success without release URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"accessJwt":  "test-access-token",
					"refreshJwt": "test-refresh-token",
					"handle":     "testuser.bsky.social",
					"did":        "did:plc:test123",
				}))
			case "/xrpc/com.atproto.repo.createRecord":
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"uri": "at://did:plc:test123/app.bsky.feed.post/test",
					"cid": "testcid",
				}))
			default:
				assert.Failf(t, "unexpected request", "unexpected request to %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(server.Close)

		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "test-project",
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					MessageTemplate: "Release {{ .ProjectName }} {{ .Tag }} is out!",
					Username:        "testuser.bsky.social",
				},
			},
		})
		ctx.Version = "v1.0.0"

		require.NoError(t, Pipe{}.Default(ctx))
		t.Setenv("BLUESKY_APP_PASSWORD", "test-password")

		pipe := Pipe{pdsURL: server.URL}
		require.NoError(t, pipe.Announce(ctx))
	})

	t.Run("login failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "AuthenticationRequired",
				"message": "Invalid credentials",
			}))
		}))
		t.Cleanup(server.Close)

		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "test-project",
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					MessageTemplate: "Release {{ .ProjectName }} {{ .Tag }}",
					Username:        "testuser.bsky.social",
				},
			},
		})
		ctx.Version = "v1.0.0"

		require.NoError(t, Pipe{}.Default(ctx))
		t.Setenv("BLUESKY_APP_PASSWORD", "wrong-password")

		pipe := Pipe{pdsURL: server.URL}
		err := pipe.Announce(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not log in to Bluesky")
	})

	t.Run("create record failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/xrpc/com.atproto.server.createSession":
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"accessJwt":  "test-access-token",
					"refreshJwt": "test-refresh-token",
					"handle":     "testuser.bsky.social",
					"did":        "did:plc:test123",
				}))
			case "/xrpc/com.atproto.repo.createRecord":
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "InvalidRequest",
					"message": "Invalid record",
				}))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(server.Close)

		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "test-project",
			Announce: config.Announce{
				Bluesky: config.Bluesky{
					MessageTemplate: "Release {{ .ProjectName }} {{ .Tag }}",
					Username:        "testuser.bsky.social",
				},
			},
		})
		ctx.Version = "v1.0.0"

		require.NoError(t, Pipe{}.Default(ctx))
		t.Setenv("BLUESKY_APP_PASSWORD", "test-password")

		pipe := Pipe{server.URL}
		err := pipe.Announce(ctx)
		require.Error(t, err)
	})
}
