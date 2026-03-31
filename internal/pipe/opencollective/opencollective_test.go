package opencollective

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.Equal(t, "opencollective", Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultTitleTemplate, ctx.Config.Announce.OpenCollective.TitleTemplate)
	require.Equal(t, defaultMessageTemplate, ctx.Config.Announce.OpenCollective.MessageTemplate)
}

func TestAnnounceInvalidTemplate(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{
				MessageTemplate: "{{ .Foo }",
			},
		},
	})

	testlib.RequireTemplateError(t, Pipe{}.Announce(ctx))
}

func TestAnnounceMissingEnv(t *testing.T) {
	t.Setenv("OPENCOLLECTIVE_TOKEN", "")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Announce: config.Announce{
			OpenCollective: config.OpenCollective{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Announce(ctx), `env: environment variable "OPENCOLLECTIVE_TOKEN" should not be empty`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.Wrap(t.Context()))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("skip empty slug", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: "true",
					Slug:    "",
				},
			},
		}))
		require.NoError(t, err)
		require.True(t, skip)
	})

	t.Run("dont skip", func(t *testing.T) {
		skip, err := Pipe{}.Skip(testctx.WrapWithCfg(t.Context(), config.Project{
			Announce: config.Announce{
				OpenCollective: config.OpenCollective{
					Enabled: "true",
					Slug:    "goreleaser",
				},
			},
		}))
		require.NoError(t, err)
		require.False(t, skip)
	})
}

func TestGraphqlResponseErr(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		r := graphqlResponse{}
		require.NoError(t, r.err())
	})

	t.Run("single error", func(t *testing.T) {
		r := graphqlResponse{Errors: []graphqlError{{Message: "not authorized"}}}
		require.EqualError(t, r.err(), "opencollective graphql error: not authorized")
	})

	t.Run("multiple errors", func(t *testing.T) {
		r := graphqlResponse{Errors: []graphqlError{
			{Message: "not authorized"},
			{Message: "invalid slug"},
		}}
		require.EqualError(t, r.err(), "opencollective graphql error: not authorized; invalid slug")
	})
}

func newTestClient(t *testing.T, handler http.HandlerFunc) client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client{endpoint: srv.URL, token: "fake-token"}
}

func TestCreateUpdateGraphqlError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"errors":[{"message":"You need to be logged in as an admin of this collective"}],"data":{"createUpdate":null}}`)
	})

	ctx := testctx.Wrap(t.Context())
	_, err := c.createUpdate(ctx, "v1.0.0", "<p>release</p>", "goreleaser")
	require.EqualError(t, err, "opencollective graphql error: You need to be logged in as an admin of this collective")
}

func TestCreateUpdateEmptyID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":{"createUpdate":{"id":""}}}`)
	})

	ctx := testctx.Wrap(t.Context())
	_, err := c.createUpdate(ctx, "v1.0.0", "<p>release</p>", "goreleaser")
	require.EqualError(t, err, "opencollective returned empty update id")
}

func TestPublishUpdateGraphqlError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"errors":[{"message":"Update not found"}]}`)
	})

	ctx := testctx.Wrap(t.Context())
	err := c.publishUpdate(ctx, "fake-id")
	require.EqualError(t, err, "opencollective graphql error: Update not found")
}

func TestNonOKStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `Unauthorized`)
	})

	ctx := testctx.Wrap(t.Context())
	_, err := c.createUpdate(ctx, "v1.0.0", "<p>release</p>", "goreleaser")
	require.ErrorContains(t, err, "incorrect response from opencollective: 401 Unauthorized")
}
