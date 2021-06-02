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
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		})
		_, err := NewGitHub(ctx, ctx.Token)

		require.NoError(t, err)
	})

	t.Run("bad api url", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		})
		_, err := NewGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "://github.mycompany.com/api": missing protocol scheme`)
	})

	t.Run("bad upload url", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "not a url:4994",
			},
		})
		_, err := NewGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "not a url:4994": first path segment in URL cannot contain colon`)
	})
}

func TestGitHubUploadReleaseIDNotInt(t *testing.T) {
	ctx := context.New(config.Project{})
	client, err := NewGitHub(ctx, ctx.Token)
	require.NoError(t, err)

	require.EqualError(
		t,
		client.Upload(ctx, "blah", &artifact.Artifact{}, nil),
		`strconv.ParseInt: parsing "blah": invalid syntax`,
	)
}

func TestGitHubReleaseURLTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		GitHubURLs: config.GitHubURLs{
			// default URL would otherwise be set via pipe/defaults
			Download: DefaultGitHubDownloadURL,
		},
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "owner",
				Name:  "name",
			},
		},
	})
	client, err := NewGitHub(ctx, ctx.Token)
	require.NoError(t, err)

	urlTpl, err := client.ReleaseURLTemplate(ctx)
	require.NoError(t, err)

	expectedURL := "https://github.com/owner/name/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
	require.Equal(t, expectedURL, urlTpl)
}

func TestGitHubCreateReleaseWrongNameTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Release: config.Release{
			NameTemplate: "{{.dddddddddd",
		},
	})
	client, err := NewGitHub(ctx, ctx.Token)
	require.NoError(t, err)

	str, err := client.CreateRelease(ctx, "")
	require.Empty(t, str)
	require.EqualError(t, err, `template: tmpl:1: unclosed action`)
}
