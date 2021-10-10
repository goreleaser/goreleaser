package defaults

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestFillBasicData(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config:    config.Project{},
	}

	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
	require.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	require.NotEmpty(t, ctx.Config.Builds)
	require.Equal(t, "goreleaser", ctx.Config.Builds[0].Binary)
	require.Equal(t, ".", ctx.Config.Builds[0].Main)
	require.Contains(t, ctx.Config.Builds[0].Goos, "darwin")
	require.Contains(t, ctx.Config.Builds[0].Goos, "linux")
	require.Contains(t, ctx.Config.Builds[0].Goarch, "386")
	require.Contains(t, ctx.Config.Builds[0].Goarch, "amd64")
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Format)
	require.Empty(t, ctx.Config.Dockers)
	require.Equal(t, "https://github.com", ctx.Config.GitHubURLs.Download)
	require.NotEmpty(t, ctx.Config.Archives[0].NameTemplate)
	require.NotEmpty(t, ctx.Config.Builds[0].Ldflags)
	require.NotEmpty(t, ctx.Config.Archives[0].Files)
	require.NotEmpty(t, ctx.Config.Dist)
}

func TestFillPartial(t *testing.T) {
	testlib.Mktmp(t)
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	ctx := &context.Context{
		Config: config.Project{
			GitHubURLs: config.GitHubURLs{
				Download: "https://github.company.com",
			},
			Dist: "disttt",
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
			Archives: []config.Archive{
				{
					Files: []config.File{
						{Source: "glob/*"},
					},
				},
			},
			Builds: []config.Build{
				{
					ID:     "build1",
					Binary: "testreleaser",
				},
				{Goos: []string{"linux"}},
				{
					ID:     "build3",
					Binary: "another",
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
			},
			Dockers: []config.Docker{
				{
					ImageTemplates: []string{"a/b"},
				},
			},
			Brews: []config.Homebrew{
				{
					Description: "foo",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Len(t, ctx.Config.Archives[0].Files, 1)
	require.NotEmpty(t, ctx.Config.Dockers[0].Goos)
	require.NotEmpty(t, ctx.Config.Dockers[0].Goarch)
	require.NotEmpty(t, ctx.Config.Dockers[0].Dockerfile)
	require.Empty(t, ctx.Config.Dockers[0].Goarm)
	require.Equal(t, "disttt", ctx.Config.Dist)
	require.NotEqual(t, "https://github.com", ctx.Config.GitHubURLs.Download)

	ctx = &context.Context{
		TokenType: context.TokenTypeGitea,

		Config: config.Project{
			GiteaURLs: config.GiteaURLs{
				API: "https://gitea.com/api/v1",
			},
		},
	}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "https://gitea.com", ctx.Config.GiteaURLs.Download)
}

func TestGiteaTemplateDownloadURL(t *testing.T) {
	tests := []struct {
		name    string
		apiURL  string
		wantErr bool
	}{
		{
			name:   "string_url",
			apiURL: "https://gitea.com/api/v1",
		},
		{
			name:   "download_url_template",
			apiURL: "{{ .Env.GORELEASER_TEST_GITEA_URLS_API }}",
		},
		{
			name:    "download_url_template_invalid_value",
			apiURL:  "{{ .Env.GORELEASER_NOT_EXISTS }}",
			wantErr: true,
		},
		{
			name:    "download_url_template_invalid",
			apiURL:  "{{.dddddddddd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitea,
			Env: context.Env{
				"GORELEASER_TEST_GITEA_URLS_API": "https://gitea.com/api/v1",
			},
			Config: config.Project{
				GiteaURLs: config.GiteaURLs{
					API: tt.apiURL,
				},
			},
		}

		err := Pipe{}.Run(ctx)
		if tt.wantErr {
			require.Error(t, err)
			return
		}

		require.NoError(t, err)
		require.Equal(t, "https://gitea.com", ctx.Config.GiteaURLs.Download)
	}
}
