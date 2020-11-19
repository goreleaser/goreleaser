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
	require.NoError(t, err)
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
