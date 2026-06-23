package client

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

// serveTestFile serves a file from testdata as the HTTP response body.
func serveTestFile(t *testing.T, w http.ResponseWriter, path string) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = io.Copy(w, f)
	require.NoError(t, err)
}

func TestClientEmpty(t *testing.T) {
	t.Parallel()
	ctx := testctx.Wrap(t.Context())
	client, err := New(ctx)
	require.Nil(t, client)
	require.EqualError(t, err, `invalid client token type: ""`)
}

func TestNewReleaseClient(t *testing.T) {
	t.Parallel()
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		cli, err := NewReleaseClient(testctx.Wrap(
			t.Context(),
			testctx.WithTokenType(context.TokenTypeGitHub),
		))
		require.NoError(t, err)
		require.IsType(t, &githubClient{}, cli)
	})

	t.Run("bad tmpl", func(t *testing.T) {
		t.Parallel()
		_, err := NewReleaseClient(testctx.WrapWithCfg(
			t.Context(),
			config.Project{
				Release: config.Release{
					Disable: "{{ .Nope }}",
				},
			},
			testctx.WithTokenType(context.TokenTypeGitHub),
		))
		testlib.RequireTemplateError(t, err)
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		cli, err := NewReleaseClient(testctx.WrapWithCfg(
			t.Context(),
			config.Project{
				Release: config.Release{
					Disable: "true",
				},
			},
			testctx.WithTokenType(context.TokenTypeGitHub),
		))
		require.NoError(t, err)
		require.IsType(t, errURLTemplater{}, cli)

		url, err := cli.ReleaseURLTemplate(nil)
		require.Empty(t, url)
		require.ErrorIs(t, err, ErrReleaseDisabled)
	})
}

func TestClientNewGitea(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API:      fakeGitea(t).URL,
			Download: "https://gitea.com",
		},
	}, testctx.GiteaTokenType)
	client, err := New(ctx)
	require.NoError(t, err)
	require.IsType(t, &giteaClient{}, client)
}

func TestClientNewGiteaInvalidURL(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "://gitea.com/api/v1",
		},
	}, testctx.GiteaTokenType)
	client, err := New(ctx)
	require.Error(t, err)
	require.Nil(t, client)
}

func TestClientNewGitLab(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "18.0.0")
	ctx := testctx.Wrap(t.Context(), testctx.GitLabTokenType)
	client, err := New(ctx)
	require.NoError(t, err)
	require.IsType(t, &gitlabClient{}, client)
}

func TestCheckBodyMaxLength(t *testing.T) {
	t.Parallel()
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, maxReleaseBodyLength)
	for i := range b {
		b[i] = letters[rand.N(len(letters))]
	}
	out := truncateReleaseBody(string(b))
	require.Len(t, out, maxReleaseBodyLength)
}

func TestTruncateReleaseBodyPreservesStart(t *testing.T) {
	t.Parallel()
	body := "A" + strings.Repeat("B", maxReleaseBodyLength)
	out := truncateReleaseBody(body)
	require.Len(t, out, maxReleaseBodyLength)
	require.True(t, strings.HasPrefix(out, "A"), "first character should be preserved")
	require.True(t, strings.HasSuffix(out, ellipsis), "truncated body should end with ellipsis")
}

func TestTruncateReleaseBodyNoTruncation(t *testing.T) {
	t.Parallel()
	body := "short body"
	out := truncateReleaseBody(body)
	require.Equal(t, body, out)
}

func TestNewIfToken(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "18.0.0")
	t.Run("valid", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.GitLabTokenType)
		client, err := New(ctx)
		require.NoError(t, err)
		require.IsType(t, &gitlabClient{}, client)

		ctx = testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"VAR=giteatoken"},
			GiteaURLs: config.GiteaURLs{
				API: fakeGitea(t).URL,
			},
		}, testctx.GiteaTokenType)
		client, err = NewIfToken(ctx, client, "{{ .Env.VAR }}")
		require.NoError(t, err)
		require.IsType(t, &giteaClient{}, client)
	})

	t.Run("empty", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.GitLabTokenType)

		client, err := New(ctx)
		require.NoError(t, err)

		client, err = NewIfToken(ctx, client, "")
		require.NoError(t, err)
		require.IsType(t, &gitlabClient{}, client)
	})

	t.Run("invalid tmpl", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.GitLabTokenType)
		_, err := NewIfToken(ctx, nil, "nope")
		require.EqualError(t, err, `expected {{ .Env.VAR_NAME }} only (no plain-text or other interpolation)`)
	})
}

func TestNewWithToken(t *testing.T) {
	t.Run("gitlab", func(t *testing.T) {
		t.Setenv("CI_SERVER_VERSION", "18.0.0")
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"TK=token"},
		}, testctx.GitLabTokenType)

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		require.IsType(t, &gitlabClient{}, cli)
	})

	t.Run("gitea", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"TK=token"},
			GiteaURLs: config.GiteaURLs{
				API: fakeGitea(t).URL,
			},
		}, testctx.GiteaTokenType)

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		require.IsType(t, &giteaClient{}, cli)
	})

	t.Run("invalid", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"TK=token"},
		}, testctx.WithTokenType(context.TokenType("nope")))
		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.EqualError(t, err, `invalid client token type: "nope"`)
		require.Nil(t, cli)
	})
}

func TestClientBlanks(t *testing.T) {
	t.Parallel()
	repo := Repo{}
	require.Empty(t, repo.String())
}

func fakeGitea(tb testing.TB) *httptest.Server {
	tb.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v1/version" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"version":"v1.11.0"}`)
			return
		}
	}))
	tb.Cleanup(srv.Close)
	return srv
}

func TestErrNoMilestoneFoundError(t *testing.T) {
	t.Parallel()
	err := ErrNoMilestoneFound{Title: "v1.0.0"}
	require.EqualError(t, err, "no milestone found: v1.0.0")
}

func TestFillDeprecated(t *testing.T) {
	t.Parallel()
	t.Run("with authors", func(t *testing.T) {
		t.Parallel()
		item := fillDeprecated(ChangelogItem{
			SHA:     "abc123",
			Message: "some message",
			Authors: []Author{{
				Name:     "John",
				Email:    "john@example.com",
				Username: "johndoe",
			}},
		})
		require.Equal(t, "John", item.AuthorName)
		require.Equal(t, "john@example.com", item.AuthorEmail)
		require.Equal(t, "johndoe", item.AuthorUsername)
	})

	t.Run("no authors", func(t *testing.T) {
		t.Parallel()
		item := fillDeprecated(ChangelogItem{
			SHA:     "abc123",
			Message: "some message",
		})
		require.Empty(t, item.AuthorName)
		require.Empty(t, item.AuthorEmail)
		require.Empty(t, item.AuthorUsername)
	})
}

func TestRepoString(t *testing.T) {
	t.Parallel()
	t.Run("with owner and name", func(t *testing.T) {
		t.Parallel()
		repo := Repo{Owner: "owner", Name: "name"}
		require.Equal(t, "owner/name", repo.String())
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		repo := Repo{}
		require.Empty(t, repo.String())
	})

	t.Run("only owner", func(t *testing.T) {
		t.Parallel()
		repo := Repo{Owner: "owner"}
		require.Equal(t, "owner/", repo.String())
	})

	t.Run("only name", func(t *testing.T) {
		t.Parallel()
		repo := Repo{Name: "name"}
		require.Equal(t, "/name", repo.String())
	})
}
