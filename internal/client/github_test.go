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
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubClient(t *testing.T) {
	t.Run("good urls", func(t *testing.T) {
		githubURL := "https://github.mycompany.com"
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    githubURL + "/api",
				Upload: githubURL + "/upload",
			},
		})

		client, err := newGitHub(ctx, ctx.Token)
		require.NoError(t, err)
		require.Equal(t, githubURL+"/api", client.client.BaseURL.String())
		require.Equal(t, githubURL+"/upload", client.client.UploadURL.String())
	})

	t.Run("bad api url", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		})
		_, err := newGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "://github.mycompany.com/api": missing protocol scheme`)
	})

	t.Run("bad upload url", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "not a url:4994",
			},
		})
		_, err := newGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "not a url:4994": first path segment in URL cannot contain colon`)
	})

	t.Run("template", func(t *testing.T) {
		githubURL := "https://github.mycompany.com"
		ctx := testctx.NewWithCfg(config.Project{
			Env: []string{
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_API=%s/api", githubURL),
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_UPLOAD=%s/upload", githubURL),
			},
			GitHubURLs: config.GitHubURLs{
				API:    "{{ .Env.GORELEASER_TEST_GITHUB_URLS_API }}",
				Upload: "{{ .Env.GORELEASER_TEST_GITHUB_URLS_UPLOAD }}",
			},
		})

		client, err := newGitHub(ctx, ctx.Token)
		require.NoError(t, err)
		require.Equal(t, githubURL+"/api", client.client.BaseURL.String())
		require.Equal(t, githubURL+"/upload", client.client.UploadURL.String())
	})

	t.Run("template invalid api", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid upload", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.Error(t, err)
	})
}

func TestGitHubUploadReleaseIDNotInt(t *testing.T) {
	ctx := testctx.New()
	client, err := newGitHub(ctx, ctx.Token)
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
			ctx := testctx.NewWithCfg(config.Project{
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
			client, err := newGitHub(ctx, ctx.Token)
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
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			NameTemplate: "{{.dddddddddd",
		},
	})
	client, err := newGitHub(ctx, ctx.Token)
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

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	b, err := client.getDefaultBranch(ctx, repo)
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

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.getDefaultBranch(ctx, repo)
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

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
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

func TestReleaseNotes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/releases/generate-notes" {
			r, err := os.Open("testdata/github/releasenotes.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	log, err := client.GenerateReleaseNotes(ctx, repo, "v1.0.0", "v1.1.0")
	require.NoError(t, err)
	require.Equal(t, "**Full Changelog**: https://github.com/someone/something/compare/v1.0.0...v1.1.0", log)
}

func TestReleaseNotesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/releases/generate-notes" {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.GenerateReleaseNotes(ctx, repo, "v1.0.0", "v1.1.0")
	require.Error(t, err)
}

func TestCloseMilestone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		t.Log(r.URL.Path)

		if r.URL.Path == "/repos/someone/something/milestones" {
			r, err := os.Open("testdata/github/milestones.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.CloseMilestone(ctx, repo, "v1.13.0"))
}

func TestOpenPullRequestHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/pulls" {
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, "main", "some title"))
}

func TestOpenPullRequestPRExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/pulls" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, "main", "some title"))
}

func TestOpenPullRequestBaseEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/pulls" {
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, "", "some title"))
}

func TestGitHubCreateFileHappyPathCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message"))
}

func TestGitHubCreateFileHappyPathUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"sha": "fake"}`)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message"))
}

func TestGitHubCreateFileFeatureBranchDoesNotExist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/branches/feature" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/repos/someone/something/git/ref/heads/main" {
			fmt.Fprint(w, `{"object": {"sha": "fake-sha"}}`)
			return
		}

		if r.URL.Path == "/repos/someone/something/git/refs" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL + "/",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "feature",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message"))
}
