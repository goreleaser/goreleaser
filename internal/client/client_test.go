package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestClientEmpty(t *testing.T) {
	ctx := &context.Context{}
	client, err := New(ctx)
	require.Nil(t, client)
	require.EqualError(t, err, `invalid client token type: ""`)
}

func TestClientNewGitea(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			GiteaURLs: config.GiteaURLs{
				// TODO: use a mocked http server to cover version api
				API:      "https://gitea.com/api/v1",
				Download: "https://gitea.com",
			},
		},
		TokenType: context.TokenTypeGitea,
		Token:     "giteatoken",
	}
	client, err := New(ctx)
	require.NoError(t, err)
	_, ok := client.(*giteaClient)
	require.True(t, ok)
}

func TestClientNewGiteaInvalidURL(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			GiteaURLs: config.GiteaURLs{
				API: "://gitea.com/api/v1",
			},
		},
		TokenType: context.TokenTypeGitea,
		Token:     "giteatoken",
	}
	client, err := New(ctx)
	require.Error(t, err)
	require.Nil(t, client)
}

func TestClientNewGitLab(t *testing.T) {
	ctx := &context.Context{
		TokenType: context.TokenTypeGitLab,
		Token:     "gitlabtoken",
	}
	client, err := New(ctx)
	require.NoError(t, err)
	_, ok := client.(*gitlabClient)
	require.True(t, ok)
}

func TestNewIfToken(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitLab,
			Token:     "gitlabtoken",
		}

		client, err := New(ctx)
		require.NoError(t, err)
		_, ok := client.(*gitlabClient)
		require.True(t, ok)

		ctx = &context.Context{
			Config: config.Project{
				GiteaURLs: config.GiteaURLs{
					API: "https://gitea.com/api/v1",
				},
			},
			TokenType: context.TokenTypeGitea,
			Token:     "giteatoken",
			Env:       map[string]string{"VAR": "token"},
		}

		client, err = NewIfToken(ctx, client, "{{ .Env.VAR }}")
		require.NoError(t, err)
		_, ok = client.(*giteaClient)
		require.True(t, ok)
	})

	t.Run("empty", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitLab,
			Token:     "gitlabtoken",
		}

		client, err := New(ctx)
		require.NoError(t, err)

		client, err = NewIfToken(ctx, client, "")
		require.NoError(t, err)
		_, ok := client.(*gitlabClient)
		require.True(t, ok)
	})

	t.Run("invalid tmpl", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitLab,
			Token:     "gitlabtoken",
		}

		_, err := NewIfToken(ctx, nil, "nope")
		require.EqualError(t, err, `expected {{ .Env.VAR_NAME }} only (no plain-text or other interpolation)`)
	})
}

func TestNewWithToken(t *testing.T) {
	t.Run("gitlab", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitLab,
			Env:       map[string]string{"TK": "token"},
		}

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		_, ok := cli.(*gitlabClient)
		require.True(t, ok)
	})

	t.Run("gitea", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitea,
			Env:       map[string]string{"TK": "token"},
			Config: config.Project{
				GiteaURLs: config.GiteaURLs{
					API: "https://gitea.com/api/v1",
				},
			},
		}

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.NoError(t, err)

		_, ok := cli.(*giteaClient)
		require.True(t, ok)
	})

	t.Run("invalid", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenType("nope"),
			Env:       map[string]string{"TK": "token"},
		}

		cli, err := newWithToken(ctx, "{{ .Env.TK }}")
		require.EqualError(t, err, `invalid client token type: "nope"`)
		require.Nil(t, cli)
	})
}

func TestClientBlanks(t *testing.T) {
	repo := Repo{}
	require.Equal(t, "", repo.String())
}
