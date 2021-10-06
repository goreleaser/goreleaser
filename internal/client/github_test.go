package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"text/template"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubClient(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		ctx := context.New(config.Project{})
		repo := Repo{
			Owner: "goreleaser",
			Name:  "goreleaser",
		}
		b, err := NewUnauthenticatedGitHub().GetDefaultBranch(ctx, repo)
		require.NoError(t, err)
		require.Equal(t, "master", b)
	})

	t.Run("good urls", func(t *testing.T) {
		githubURL := "https://github.mycompany.com"
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    githubURL + "/api",
				Upload: githubURL + "/upload",
			},
		})

		client, err := NewGitHub(ctx, ctx.Token)
		require.NoError(t, err)

		githubClient, ok := client.(*githubClient)
		require.True(t, ok)
		require.Equal(t, githubURL+"/api", githubClient.client.BaseURL.String())
		require.Equal(t, githubURL+"/upload", githubClient.client.UploadURL.String())
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

	t.Run("template", func(t *testing.T) {
		githubURL := "https://github.mycompany.com"
		ctx := context.New(config.Project{
			Env: []string{
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_API=%s/api", githubURL),
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_UPLOAD=%s/upload", githubURL),
			},
			GitHubURLs: config.GitHubURLs{
				API:    "{{ .Env.GORELEASER_TEST_GITHUB_URLS_API }}",
				Upload: "{{ .Env.GORELEASER_TEST_GITHUB_URLS_UPLOAD }}",
			},
		})

		client, err := NewGitHub(ctx, ctx.Token)
		require.NoError(t, err)

		githubClient, ok := client.(*githubClient)
		require.True(t, ok)
		require.Equal(t, githubURL+"/api", githubClient.client.BaseURL.String())
		require.Equal(t, githubURL+"/upload", githubClient.client.UploadURL.String())
	})

	t.Run("template invalid api", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := NewGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid upload", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := NewGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid", func(t *testing.T) {
		ctx := context.New(config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := NewGitHub(ctx, ctx.Token)
		require.Error(t, err)
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
	tests := []struct {
		name            string
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "default_download_url",
			downloadURL:     DefaultGitHubDownloadURL,
			wantDownloadURL: "https://github.com/owner/name/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITHUB_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://github.mycompany.com/owner/name/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
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
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.New(config.Project{
				Env: []string{
					"GORELEASER_TEST_GITHUB_URLS_DOWNLOAD=https://github.mycompany.com",
				},
				GitHubURLs: config.GitHubURLs{
					Download: tt.downloadURL,
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
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantDownloadURL, urlTpl)
		})
	}
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

func TestGithubGetDefaultBranch(t *testing.T) {
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"default_branch": "main"}`)
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})

	client, err := NewGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	b, err := client.GetDefaultBranch(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, "main", b)
	require.Equal(t, 1, totalRequests)
}

func TestGithubGetDefaultBranchErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := NewGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.GetDefaultBranch(ctx, repo)
	require.Error(t, err)
}

func TestChangelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/compare/v1.0.0...v1.1.0" {
			r, err := os.Open("testdata/github/compare.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := NewGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	log, err := client.Changelog(ctx, repo, "v1.0.0", "v1.1.0")
	require.NoError(t, err)
	require.Equal(t, "6dcb09b5b57875f334f61aebed695e2e4193db5e: Fix all the bugs (@octocat)", log)
}
