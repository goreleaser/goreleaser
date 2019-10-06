package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubClient(t *testing.T) {
	t.Run("good urls", func(t *testing.T) {
		_, err := NewGitHub(context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		}))

		require.NoError(t, err)
	})

	t.Run("bad api url", func(t *testing.T) {
		_, err := NewGitHub(context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		}))

		require.EqualError(t, err, "parse ://github.mycompany.com/api: missing protocol scheme")
	})

	t.Run("bad upload url", func(t *testing.T) {
		_, err := NewGitHub(context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "not a url:4994",
			},
		}))

		require.EqualError(t, err, "parse not a url:4994: first path segment in URL cannot contain colon")
	})
}

func TestGitHubUploadReleaseIDNotInt(t *testing.T) {
	var ctx = context.New(config.Project{})
	client, err := NewGitHub(ctx)
	require.NoError(t, err)

	require.EqualError(
		t,
		client.Upload(ctx, "blah", &artifact.Artifact{}, nil),
		`strconv.ParseInt: parsing "blah": invalid syntax`,
	)
}

func TestGitHubCreateReleaseWrongNameTemplate(t *testing.T) {
	var ctx = context.New(config.Project{
		Release: config.Release{
			NameTemplate: "{{.dddddddddd",
		},
	})
	client, err := NewGitHub(ctx)
	require.NoError(t, err)

	str, err := client.CreateRelease(ctx, "")
	require.Empty(t, str)
	require.EqualError(t, err, `template: tmpl:1: unclosed action`)
}
