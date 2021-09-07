package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestGitLabReleaseURLTemplate(t *testing.T) {
	tests := []struct {
		name            string
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "default_download_url",
			downloadURL:     DefaultGitLabDownloadURL,
			wantDownloadURL: "https://gitlab.com/owner/name/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITLAB_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://gitlab.mycompany.com/owner/name/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}",
		},
		{
			name:        "download_url_template_invalid_value",
			downloadURL: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			wantErr:     true,
		},
		{
			name:        "download_url_template_invalid",
			downloadURL: "{{.dddddddddd",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		ctx := context.New(config.Project{
			Env: []string{
				"GORELEASER_TEST_GITLAB_URLS_DOWNLOAD=https://gitlab.mycompany.com",
			},
			GitLabURLs: config.GitLabURLs{
				Download: tt.downloadURL,
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
		if tt.wantErr {
			require.Error(t, err)
			return
		}

		require.NoError(t, err)
		require.Equal(t, tt.wantDownloadURL, urlTpl)
	}
}

func TestGitLabURLsAPITemplate(t *testing.T) {
	tests := []struct {
		name     string
		apiURL   string
		wantHost string
	}{
		{
			name:     "default_values",
			wantHost: "gitlab.com",
		},
		{
			name:     "speicifed_api_env_key",
			apiURL:   "https://gitlab.mycompany.com",
			wantHost: "gitlab.mycompany.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envs := []string{}
			gitlabURLs := config.GitLabURLs{}

			if tt.apiURL != "" {
				envs = append(envs, fmt.Sprintf("GORELEASER_TEST_GITLAB_URLS_API=%s", tt.apiURL))
				gitlabURLs.API = "{{ .Env.GORELEASER_TEST_GITLAB_URLS_API }}"
			}

			ctx := context.New(config.Project{
				Env:        envs,
				GitLabURLs: gitlabURLs,
			})

			client, err := NewGitLab(ctx, ctx.Token)
			require.NoError(t, err)

			gitlabClient, ok := client.(*gitlabClient)
			require.True(t, ok)

			require.Equal(t, tt.wantHost, gitlabClient.client.BaseURL().Host)
		})
	}

	t.Run("no_env_specified", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := NewGitLab(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("invalid_template", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := NewGitLab(ctx, ctx.Token)
		require.Error(t, err)
	})
}

func TestGitLabURLsDownloadTemplate(t *testing.T) {
	tests := []struct {
		name        string
		downloadURL string
		wantURL     string
		wantErr     bool
	}{
		{
			name:    "empty_download_url",
			wantURL: "/",
		},
		{
			name:        "download_url_template",
			downloadURL: "{{ .Env.GORELEASER_TEST_GITLAB_URLS_DOWNLOAD }}",
			wantURL:     "https://gitlab.mycompany.com/",
		},
		{
			name:        "download_url_template_invalid_value",
			downloadURL: "{{ .Eenv.GORELEASER_NOT_EXISTS }}",
			wantErr:     true,
		},
		{
			name:        "download_url_template_invalid",
			downloadURL: "{{.dddddddddd",
			wantErr:     true,
		},
		{
			name:        "download_url_string",
			downloadURL: "https://gitlab.mycompany.com",
			wantURL:     "https://gitlab.mycompany.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer fmt.Fprint(w, "{}")
				defer w.WriteHeader(http.StatusOK)
				defer r.Body.Close()

				if !strings.Contains(r.URL.Path, "assets/links") {
					_, _ = io.Copy(io.Discard, r.Body)
					return
				}

				b, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				reqBody := map[string]interface{}{}
				err = json.Unmarshal(b, &reqBody)
				require.NoError(t, err)

				require.Equal(t, tt.wantURL, reqBody["url"])
			}))
			defer srv.Close()

			ctx := context.New(config.Project{
				Env: []string{
					"GORELEASER_TEST_GITLAB_URLS_DOWNLOAD=https://gitlab.mycompany.com",
				},
				Release: config.Release{
					GitLab: config.Repo{
						Owner: "test",
						Name:  "test",
					},
				},
				GitLabURLs: config.GitLabURLs{
					API:      srv.URL,
					Download: tt.downloadURL,
				},
			})

			tmpFile, err := os.CreateTemp(t.TempDir(), "")
			require.NoError(t, err)

			client, err := NewGitLab(ctx, ctx.Token)
			require.NoError(t, err)

			err = client.Upload(ctx, "1234", &artifact.Artifact{Name: "test", Path: "some-path"}, tmpFile)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
