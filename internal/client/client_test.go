package client

import (
	"math/rand"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestClientEmpty(t *testing.T) {
	ctx := testctx.New()
	client, err := New(ctx)
	require.Nil(t, client)
	require.EqualError(t, err, `invalid client token type: ""`)
}

func TestClientNewGitea(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		GiteaURLs: config.GiteaURLs{
			// TODO: use a mocked http server to cover version api
			API:      "https://gitea.com/api/v1",
			Download: "https://gitea.com",
		},
	}, testctx.GiteaTokenType)
	client, err := New(ctx)
	require.NoError(t, err)
	_, ok := client.(*giteaClient)
	require.True(t, ok)
}

func TestClientNewGiteaInvalidURL(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "://gitea.com/api/v1",
		},
	}, testctx.GiteaTokenType)
	client, err := New(ctx)
	require.Error(t, err)
	require.Nil(t, client)
}

func TestClientNewGitLab(t *testing.T) {
	ctx := testctx.New(testctx.GitLabTokenType)
	client, err := New(ctx)
	require.NoError(t, err)
	_, ok := client.(*gitlabClient)
	require.True(t, ok)
}

func TestCheckBodyMaxLength(t *testing.T) {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, maxReleaseBodyLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	out := truncateReleaseBody(string(b))
	require.Len(t, out, maxReleaseBodyLength)
}

func TestNewIfToken(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ctx := testctx.New(testctx.GitLabTokenType)
		client, err := New(ctx)
		require.NoError(t, err)
		_, ok := client.(*gitlabClient)
		require.True(t, ok)

		ctx = testctx.NewWithCfg(config.Project{
			Env: []string{"VAR=giteatoken"},
			GiteaURLs: config.GiteaURLs{
				API: "https://gitea.com/api/v1",
			},
		}, testctx.GiteaTokenType)
		client, err = NewIfToken(ctx, client, "{{ .Env.VAR }}")
		require.NoError(t, err)
		_, ok = client.(*giteaClient)
		require.True(t, ok)
	})

	t.Run("empty", func(t *testing.T) {
		ctx := testctx.New(testctx.GitLabTokenType)

		client, err := New(ctx)
		require.NoError(t, err)

		client, err = NewIfToken(ctx, client, "")
		require.NoError(t, err)
		_, ok := client.(*gitlabClient)
		require.True(t, ok)
	})

	t.Run("invalid tmpl", func(t *testing.T) {
		ctx := testctx.New(testctx.GitLabTokenType)
		_, err := NewIfToken(ctx, nil, "nope")
		require.EqualError(t, err, `expected {{ .Env.VAR_NAME }} only (no plain-text or other interpolation)`)
	})
}

func TestNewWithToken(t *testing.T) {
	t.Run("gitlab", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"TK=token"},
		}, testctx.GitLabTokenType)

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		_, ok := cli.(*gitlabClient)
		require.True(t, ok)
	})

	t.Run("gitea", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"TK=token"},
			GiteaURLs: config.GiteaURLs{
				API: "https://gitea.com/api/v1",
			},
		}, testctx.GiteaTokenType)

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		_, ok := cli.(*giteaClient)
		require.True(t, ok)
	})

	t.Run("invalid", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{"TK=token"},
		}, testctx.WithTokenType(context.TokenType("nope")))
		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.EqualError(t, err, `invalid client token type: "nope"`)
		require.Nil(t, cli)
	})
}

func TestClientBlanks(t *testing.T) {
	repo := Repo{}
	require.Equal(t, "", repo.String())
}
