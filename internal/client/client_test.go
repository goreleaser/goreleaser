package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestClientEmpty(t *testing.T) {
	ctx := &context.Context{}
	client, err := New(ctx)
	assert.Nil(t, client)
	assert.NoError(t, err)
}

func TestClientNewGitea(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			GiteaURLs: config.GiteaURLs{
				API: "https://git.dtluna.net/api/v1",
			},
		},
		TokenType: context.TokenTypeGitea,
		Token:     "giteatoken",
	}
	client, err := New(ctx)
	assert.NoError(t, err)
	_, ok := client.(*giteaClient)
	assert.True(t, ok)
}

func TestClientNewGiteaInvalidURL(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			GiteaURLs: config.GiteaURLs{
				API: "://git.dtluna.net/api/v1",
			},
		},
		TokenType: context.TokenTypeGitea,
		Token:     "giteatoken",
	}
	client, err := New(ctx)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClientNewGitLab(t *testing.T) {
	ctx := &context.Context{
		TokenType: context.TokenTypeGitLab,
		Token:     "gitlabtoken",
	}
	client, err := New(ctx)
	assert.NoError(t, err)
	_, ok := client.(*gitlabClient)
	assert.True(t, ok)
}
