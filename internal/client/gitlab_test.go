package client

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestGitLabReleaseURLTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		GitLabURLs: config.GitLabURLs{
			// default URL would otherwise be set via pipe/defaults
			Download: DefaultGitLabDownloadURL,
		},
		Release: config.Release{
			GitLab: config.Repo{
				Owner: "owner",
				Name:  "name",
			},
		},
	})
	client, err := NewGitLab(ctx, ctx.Token)
	require.NoError(t, err)

	urlTpl, err := client.ReleaseURLTemplate(ctx)
	require.NoError(t, err)

	expectedUrl := "https://gitlab.com/owner/name/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}"
	require.Equal(t, expectedUrl, urlTpl)
}
