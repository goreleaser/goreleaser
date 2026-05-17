package client

import (
	stdctx "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubClient(t *testing.T) {
	t.Parallel()
	t.Run("good urls", func(t *testing.T) {
		t.Parallel()
		githubURL := "https://github.mycompany.com"
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    githubURL + "/api/v3",
				Upload: githubURL,
			},
		})

		client, err := newGitHub(ctx, ctx.Token)
		require.NoError(t, err)
		require.Equal(t, githubURL+"/api/v3/", client.client.BaseURL.String())
		require.Equal(t, githubURL+"/api/uploads/", client.client.UploadURL.String())
	})

	t.Run("good urls ending with /", func(t *testing.T) {
		t.Parallel()
		githubURL := "https://github.mycompany.com"
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    githubURL + "/api/v3/",
				Upload: githubURL + "/api/uploads/",
			},
		})

		client, err := newGitHub(ctx, ctx.Token)
		require.NoError(t, err)
		require.Equal(t, githubURL+"/api/v3/", client.client.BaseURL.String())
		require.Equal(t, githubURL+"/api/uploads/", client.client.UploadURL.String())
	})

	t.Run("bad api url", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "://github.mycompany.com/api",
				Upload: "https://github.mycompany.com/upload",
			},
		})
		_, err := newGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "://github.mycompany.com/api": missing protocol scheme`)
	})

	t.Run("bad upload url", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "not a url:4994",
			},
		})
		_, err := newGitHub(ctx, ctx.Token)

		require.EqualError(t, err, `parse "not a url:4994": first path segment in URL cannot contain colon`)
	})

	t.Run("template", func(t *testing.T) {
		t.Parallel()
		githubURL := "https://github.mycompany.com"
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_API=%s", githubURL),
				fmt.Sprintf("GORELEASER_TEST_GITHUB_URLS_UPLOAD=%s", githubURL),
			},
			GitHubURLs: config.GitHubURLs{
				API:    "{{ .Env.GORELEASER_TEST_GITHUB_URLS_API }}",
				Upload: "{{ .Env.GORELEASER_TEST_GITHUB_URLS_UPLOAD }}",
			},
		})

		client, err := newGitHub(ctx, ctx.Token)
		require.NoError(t, err)
		require.Equal(t, githubURL+"/api/v3/", client.client.BaseURL.String())
		require.Equal(t, githubURL+"/api/uploads/", client.client.UploadURL.String())
	})

	t.Run("template invalid api", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid upload", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API:    "https://github.mycompany.com/api",
				Upload: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("template invalid", func(t *testing.T) {
		t.Parallel()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := newGitHub(ctx, ctx.Token)
		require.Error(t, err)
	})
}

func TestGitHubUploadReleaseIDNotInt(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(_ http.ResponseWriter, _ *http.Request) {
	})
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	require.EqualError(
		t,
		client.Upload(ctx, "blah", &artifact.Artifact{}),
		`strconv.ParseInt: parsing "blah": invalid syntax`,
	)
}

func TestGitHubReleaseURLTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "default_download_url",
			downloadURL:     DefaultGitHubDownloadURL,
			wantDownloadURL: "https://github.com/owner/name/releases/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITHUB_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://github.mycompany.com/owner/name/releases/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}",
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
			t.Parallel()
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			NameTemplate: "{{.dddddddddd",
		},
	})
	client, err := newGitHub(ctx, ctx.Token)
	require.NoError(t, err)

	str, err := client.CreateRelease(ctx, "")
	require.Empty(t, str)
	testlib.RequireTemplateError(t, err)
}

func TestGitHubGetDefaultBranch(t *testing.T) {
	t.Parallel()
	totalRequests := 0
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"default_branch": "main"}`)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubGetDefaultBranchErr(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "{}")
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Retry: config.Retry{Attempts: 1},
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

func TestGitHubChangelog(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/compare/v1.0.0...v1.1.0" {
			serveTestFile(t, w, "testdata/github/compare.json")
			return
		}
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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
	require.Equal(t, []ChangelogItem{
		{
			SHA:     "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			Message: "Fix all the bugs",
			Authors: []Author{{
				Name:     "Octocat",
				Email:    "octo@cat",
				Username: "octocat",
			}},
			AuthorName:     "Octocat",
			AuthorEmail:    "octo@cat",
			AuthorUsername: "octocat",
		},
	}, log)
}

func TestGitHubReleaseNotes(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/releases/generate-notes" {
			serveTestFile(t, w, "testdata/github/releasenotes.json")
			return
		}
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubReleaseNotesError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/releases/generate-notes" {
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubCloseMilestone(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/milestones" {
			serveTestFile(t, w, "testdata/github/milestones.json")
			return
		}
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

const testPRTemplate = "fake template\n- [ ] mark this\n---"

func TestGitHubOpenPullRequestCrossRepo(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			content := github.RepositoryContent{
				Encoding: new("base64"),
				Content:  new(base64.StdEncoding.EncodeToString([]byte(testPRTemplate))),
			}
			bts, _ := json.Marshal(content)
			_, _ = w.Write(bts)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr github.NewPullRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.GetBase())
			assert.Equal(t, "someoneelse:something:foo", pr.GetHead())
			assert.Equal(t, testPRTemplate+"\n"+prFooter, pr.GetBody())
			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	base := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}
	head := Repo{
		Owner:  "someoneelse",
		Name:   "something",
		Branch: "foo",
	}
	require.NoError(t, client.OpenPullRequest(ctx, base, head, "some title", false))
}

func TestGitHubOpenPullRequestHappyPath(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			content := github.RepositoryContent{
				Encoding: new("base64"),
				Content:  new(base64.StdEncoding.EncodeToString([]byte(testPRTemplate))),
			}
			bts, _ := json.Marshal(content)
			_, _ = w.Write(bts)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
}

func TestGitHubOpenPullRequestNoBaseBranchDraft(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr github.NewPullRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.GetBase())
			assert.Equal(t, "someone:something:foo", pr.GetHead())
			assert.True(t, pr.GetDraft())

			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{
		Branch: "foo",
	}, "some title", true))
}

func TestGitHubOpenPullRequestPRExists(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
}

func TestGitHubOpenPullRequestBaseEmpty(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "foo",
	}

	require.NoError(t, client.OpenPullRequest(ctx, Repo{}, repo, "some title", false))
}

func TestGitHubOpenPullRequestHeadEmpty(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/pulls" {
			serveTestFile(t, w, "testdata/github/pull.json")
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
}

func TestGitHubCreateFileHappyPathCreate(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			var data github.RepositoryContentFileOptions
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&data))
			assert.Nil(t, data.SHA)
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"sha": "fake"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			var data github.RepositoryContentFileOptions
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&data))
			assert.Equal(t, "fake", data.GetSHA())
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubCreateFileFeatureBranchAlreadyExists(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/branches/feature" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/git/ref/heads/main" {
			fmt.Fprint(w, `{"object": {"sha": "fake-sha"}}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/git/refs" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprintf(w, `{"message": "Reference already exists"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubCreateFileFeatureBranchDoesNotExist(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/branches/feature" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/git/ref/heads/main" {
			fmt.Fprint(w, `{"object": {"sha": "fake-sha"}}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/git/refs" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
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

func TestGitHubCreateFileFeatureBranchNilObject(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something/branches/feature" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/git/ref/heads/main" {
			// Return ref with nil object
			fmt.Fprint(w, `{}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "feature",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not create ref")
	require.Contains(t, err.Error(), "sha must be provided")
}

func TestGitHubCheckRateLimit(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	reset := now.Add(1392 * time.Millisecond)
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/rate_limit" {
			called.Store(true)
			w.WriteHeader(http.StatusOK)
			resetstr, _ := github.Timestamp{Time: reset}.MarshalJSON()
			fmt.Fprintf(w, `{"resources":{"core":{"remaining":98,"reset":%s}}}`, string(resetstr))
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	short, cancel := stdctx.WithTimeout(t.Context(), 250*time.Millisecond)
	t.Cleanup(cancel)
	ctx := testctx.WrapWithCfg(short, config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	client.checkRateLimit(ctx)

	require.True(t, called.Load(), "should have checked rate limit")
}

func TestGitHubCreateRelease(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/tags/v1.0.0" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodPost {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.JSONEq(t, `{"name": "v1.0.0", "tag_name": "v1.0.0", "target_commitish": "test", "body": "test release", "draft": true, "prerelease": false}`, string(got))

			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id": 1, "html_url": "https://github.com/goreleaser/test/releases/v1.0.0"}`)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate: "v1.0.0",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
				TargetCommitish: "test",
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	release, err := client.CreateRelease(ctx, "test release")
	require.NoError(t, err)
	require.Equal(t, "1", release)
}

func TestGitHubCreateReleaseDeleteExistingDraft(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			serveTestFile(t, w, "testdata/github/releases.json")
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/tags/v1.0.0" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id": 2, "html_url": "https://github.com/goreleaser/test/releases/v1.0.0"}`)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate: "v1.0.0",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
				Draft:                true,
				ReplaceExistingDraft: true,
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	release, err := client.CreateRelease(ctx, "test draft release")
	require.NoError(t, err)
	require.Equal(t, "2", release)
}

func TestGitHubCreateReleaseUpdateExisting(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/tags/v1.0.0" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": 3, "name": "v1.0.0", "body": "This is an existing release"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/3" && r.Method == http.MethodPatch {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.JSONEq(t, `{"name": "v1.0.0", "tag_name": "v1.0.0", "body": "This is an existing release", "prerelease": false}`, string(got))

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": 3, "name": "v1.0.0", "body": "This is an existing release"}`)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate: "v1.0.0",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	release, err := client.CreateRelease(ctx, "test update release")
	require.NoError(t, err)
	require.Equal(t, "3", release)
}

func TestGitHubCreateReleaseUseExistingDraft(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			serveTestFile(t, w, "testdata/github/releases.json")
			return
		}

		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/1" && r.Method == http.MethodPatch {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.JSONEq(t, `{"name": "v1.0.0", "tag_name": "v1.0.0", "body": "Existing draft release", "draft": true, "prerelease": false}`, string(got))

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": 1, "name": "v1.0.0"}`)
			return
		}

		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate: "v1.0.0",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
				UseExistingDraft: true,
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	release, err := client.CreateRelease(ctx, "test update draft release")
	require.NoError(t, err)
	require.Equal(t, "1", release)
}

func TestGitHubCreateFileWithGitHubAppToken(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			// Verify that committer is not set in the request
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var reqData map[string]any
			assert.NoError(t, json.Unmarshal(body, &reqData))

			// Verify committer is not present when using GitHub App token
			_, hasCommitter := reqData["committer"]
			assert.False(t, hasCommitter, "committer should not be set when using GitHub App token")

			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{
		UseGitHubAppToken: true,
	}, repo, []byte("content"), "file.txt", "message"))
}

func TestGitHubCreateFileWithoutGitHubAppToken(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v3/repos/someone/something" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			// Verify that committer is set in the request
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var reqData map[string]any
			assert.NoError(t, json.Unmarshal(body, &reqData))

			// Verify committer is present when not using GitHub App token
			committer, hasCommitter := reqData["committer"]
			assert.True(t, hasCommitter, "committer should be set when not using GitHub App token")

			committerMap := committer.(map[string]any)
			assert.Equal(t, "test-author", committerMap["name"])
			assert.Equal(t, "test@example.com", committerMap["email"])

			w.WriteHeader(http.StatusOK)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{
		Name:  "test-author",
		Email: "test@example.com",
	}, repo, []byte("content"), "file.txt", "message"))
}

func TestGitHubAuthorsLookup(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Retry: config.Retry{Attempts: 1},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	t.Run("noreply email without numeric id", func(t *testing.T) {
		result := client.authorsLookup([]Author{
			{Name: "Foo Bar", Email: "foobar@users.noreply.github.com"},
		})
		require.Equal(t, "foobar", result[0].Username)
	})

	t.Run("noreply email with numeric id", func(t *testing.T) {
		result := client.authorsLookup([]Author{
			{Name: "Foo Bar", Email: "12345+foobar@users.noreply.github.com"},
		})
		require.Equal(t, "foobar", result[0].Username)
	})

	t.Run("non-noreply email is left alone", func(t *testing.T) {
		result := client.authorsLookup([]Author{
			{Name: "Some User", Email: "someone@example.com"},
		})
		require.Empty(t, result[0].Username)
	})

	t.Run("mixed authors", func(t *testing.T) {
		result := client.authorsLookup([]Author{
			{Name: "Noreply User", Email: "noreplyuser@users.noreply.github.com"},
			{Name: "Noreply Plus", Email: "999+noreplyplus@users.noreply.github.com"},
			{Name: "Regular User", Email: "regular@example.com"},
		})
		require.Equal(t, "noreplyuser", result[0].Username)
		require.Equal(t, "noreplyplus", result[1].Username)
		require.Empty(t, result[2].Username)
	})
}

func TestGitHubPublishRelease(t *testing.T) {
	t.Parallel()

	t.Run("draft stays draft", func(t *testing.T) {
		t.Parallel()
		srv := githubTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				GitHub: config.Repo{Owner: "owner", Name: "name"},
				Draft:  true,
			},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		require.NoError(t, client.PublishRelease(ctx, "123"))
	})

	t.Run("happy path publish", func(t *testing.T) {
		t.Parallel()
		var requestBody string
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/owner/name/releases/123" && r.Method == http.MethodPatch {
				bts, _ := io.ReadAll(r.Body)
				requestBody = string(bts)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"id":123,"html_url":"https://github.com/owner/name/releases/tag/v1.0.0"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				GitHub:     config.Repo{Owner: "owner", Name: "name"},
				Draft:      false,
				MakeLatest: "true",
			},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		require.NoError(t, client.PublishRelease(ctx, "123"))
		require.Contains(t, requestBody, `"draft":false`)
	})

	t.Run("publish prerelease", func(t *testing.T) {
		t.Parallel()
		var requestBody string
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/owner/name/releases/123" && r.Method == http.MethodPatch {
				bts, _ := io.ReadAll(r.Body)
				requestBody = string(bts)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"id":123,"html_url":"https://github.com/owner/name/releases/tag/v1.0.0"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				NameTemplate: "custom title",
				GitHub:       config.Repo{Owner: "owner", Name: "name"},
				Draft:        false,
				MakeLatest:   "true",
			},
		})
		ctx.PreRelease = true
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		require.NoError(t, client.PublishRelease(ctx, "123"))
		require.Contains(t, requestBody, `"name":"custom title"`)
		require.Contains(t, requestBody, `"prerelease":true`)
		require.Contains(t, requestBody, `"make_latest":"false"`)
	})

	t.Run("with make latest", func(t *testing.T) {
		t.Parallel()
		var requestBody string
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/owner/name/releases/123" && r.Method == http.MethodPatch {
				bts, _ := io.ReadAll(r.Body)
				requestBody = string(bts)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"id":123,"html_url":"https://github.com/owner/name/releases/tag/v1.0.0"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				GitHub:     config.Repo{Owner: "owner", Name: "name"},
				Draft:      false,
				MakeLatest: "true",
			},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		require.NoError(t, client.PublishRelease(ctx, "123"))
		require.Contains(t, requestBody, `"make_latest":"true"`)
	})

	t.Run("with discussion category", func(t *testing.T) {
		t.Parallel()
		var requestBody string
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/owner/name/releases/123" && r.Method == http.MethodPatch {
				bts, _ := io.ReadAll(r.Body)
				requestBody = string(bts)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"id":123,"html_url":"https://github.com/owner/name/releases/tag/v1.0.0"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				GitHub:                 config.Repo{Owner: "owner", Name: "name"},
				Draft:                  false,
				DiscussionCategoryName: "General",
			},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		require.NoError(t, client.PublishRelease(ctx, "123"))
		require.Contains(t, requestBody, `"discussion_category_name":"General"`)
	})

	t.Run("bad release id", func(t *testing.T) {
		t.Parallel()
		srv := githubTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
			Release: config.Release{
				GitHub: config.Repo{Owner: "owner", Name: "name"},
				Draft:  false,
			},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		err = client.PublishRelease(ctx, "not-a-number")
		require.Error(t, err)
		require.Contains(t, err.Error(), "non-numeric release ID")
	})
}

func TestGitHubPublishReleaseBadMakeLatestTemplate(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{API: srv.URL},
		Release: config.Release{
			GitHub:     config.Repo{Owner: "owner", Name: "name"},
			Draft:      false,
			MakeLatest: "{{ .Env.NOPE }}",
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	err = client.PublishRelease(ctx, "123")
	require.Error(t, err)
	require.Contains(t, err.Error(), "templating GitHub make_latest")
}

func TestGitHubSyncFork(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/headowner/headname/merge-upstream" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"merge_type":"fast-forward","base_branch":"main","message":"Successfully fetched"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		err = client.SyncFork(
			ctx,
			Repo{Owner: "headowner", Name: "headname"},
			Repo{Owner: "baseowner", Name: "basename", Branch: "main"},
		)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/headowner/headname/merge-upstream" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusUnprocessableEntity)
				fmt.Fprint(w, `{"message":"merge conflict"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		err = client.SyncFork(
			ctx,
			Repo{Owner: "headowner", Name: "headname"},
			Repo{Owner: "baseowner", Name: "basename", Branch: "main"},
		)
		require.Error(t, err)
	})

	t.Run("no base branch uses default", func(t *testing.T) {
		t.Parallel()
		srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.URL.Path == "/api/v3/repos/baseowner/basename" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"default_branch":"develop"}`)
				return
			}
			if r.URL.Path == "/api/v3/repos/headowner/headname/merge-upstream" && r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"merge_type":"fast-forward","base_branch":"develop","message":"Successfully fetched"}`)
				return
			}
			t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
		})
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitHubURLs: config.GitHubURLs{API: srv.URL},
		})
		client, err := newGitHub(ctx, "test-token")
		require.NoError(t, err)
		err = client.SyncFork(
			ctx,
			Repo{Owner: "headowner", Name: "headname"},
			Repo{Owner: "baseowner", Name: "basename"},
		)
		require.NoError(t, err)
	})
}

func TestGitHubCloseMilestoneNotFound(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/milestones" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[]`)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}
	err = client.CloseMilestone(ctx, repo, "v9.9.9")
	require.ErrorAs(t, err, &ErrNoMilestoneFound{})
}

func TestGitHubUploadReplaceExisting(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasSuffix(r.URL.Path, "/releases/123/assets") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"message":"already exists"}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/owner/name/releases/123/assets" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":456,"name":"test-file.txt"}]`)
			return
		}
		if r.URL.Path == "/api/v3/repos/owner/name/releases/assets/456" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API:    srv.URL,
			Upload: srv.URL,
		},
		Release: config.Release{
			GitHub:                   config.Repo{Owner: "owner", Name: "name"},
			ReplaceExistingArtifacts: true,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	f, err := os.CreateTemp(t.TempDir(), "upload-test")
	require.NoError(t, err)
	fmt.Fprint(f, "test content")
	require.NoError(t, f.Close())
	err = client.Upload(ctx, "123", &artifact.Artifact{Name: "test-file.txt", Path: f.Name()})
	require.Error(t, err)
}

func TestGitHubUploadNoReplace(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasSuffix(r.URL.Path, "/releases/123/assets") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"message":"already exists"}`)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API:    srv.URL,
			Upload: srv.URL,
		},
		Release: config.Release{
			GitHub:                   config.Repo{Owner: "owner", Name: "name"},
			ReplaceExistingArtifacts: false,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	f, err := os.CreateTemp(t.TempDir(), "upload-test")
	require.NoError(t, err)
	fmt.Fprint(f, "test content")
	require.NoError(t, f.Close())
	err = client.Upload(ctx, "123", &artifact.Artifact{Name: "test-file.txt", Path: f.Name()})
	require.Error(t, err)
}

func TestHeadString(t *testing.T) {
	t.Parallel()

	t.Run("head wins over base", func(t *testing.T) {
		t.Parallel()
		result := headString(
			Repo{Owner: "base-owner", Name: "base-name", Branch: "base-branch"},
			Repo{Owner: "head-owner", Name: "head-name", Branch: "head-branch"},
		)
		require.Equal(t, "head-owner:head-name:head-branch", result)
	})

	t.Run("base used when head empty", func(t *testing.T) {
		t.Parallel()
		result := headString(
			Repo{Owner: "base-owner", Name: "base-name", Branch: "base-branch"},
			Repo{},
		)
		require.Equal(t, "base-owner:base-name:base-branch", result)
	})

	t.Run("mixed", func(t *testing.T) {
		t.Parallel()
		result := headString(
			Repo{Owner: "base-owner", Name: "base-name", Branch: "base-branch"},
			Repo{Owner: "head-owner"},
		)
		require.Equal(t, "head-owner:base-name:base-branch", result)
	})
}

func TestBodyOf(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "no response", bodyOf(nil))
	})

	t.Run("nil body", func(t *testing.T) {
		t.Parallel()
		resp := &github.Response{
			Response: &http.Response{},
		}
		require.Equal(t, "no response", bodyOf(resp))
	})

	t.Run("with body content", func(t *testing.T) {
		t.Parallel()
		resp := &github.Response{
			Response: &http.Response{
				Body: io.NopCloser(strings.NewReader("hello world")),
			},
		}
		require.Equal(t, "hello world", bodyOf(resp))
	})
}

func TestGitHubChangelogError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/compare/v1.0.0...v1.1.0" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	result, err := client.Changelog(ctx, repo, "v1.0.0", "v1.1.0")
	require.Error(t, err)
	require.Nil(t, result)
}

func TestGitHubCloseMilestoneError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/milestones" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	err = client.CloseMilestone(ctx, repo, "v1.0.0")
	require.Error(t, err)
}

func TestGitHubOpenPullRequestDefaultBranchError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.OpenPullRequest(
		ctx,
		Repo{Owner: "someone", Name: "something"},
		Repo{Branch: "foo"},
		"some title", false,
	)
	require.Error(t, err)
}

func TestGitHubOpenPullRequest422(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/pulls" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"message":"Validation Failed"}`)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.OpenPullRequest(
		ctx,
		Repo{Owner: "someone", Name: "something", Branch: "main"},
		Repo{Branch: "foo"},
		"some title", false,
	)
	require.NoError(t, err)
}

func TestGitHubSyncForkDefaultBranchError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/baseowner/basename" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.SyncFork(
		ctx,
		Repo{Owner: "headowner", Name: "headname"},
		Repo{Owner: "baseowner", Name: "basename"},
	)
	require.Error(t, err)
}

func TestGitHubCreateFileDefaultBranchError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not get default branch")
}

func TestGitHubCreateFileBranchNotFoundCreatesRef(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/branches/newbranch" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/git/ref/heads/main" && r.Method == http.MethodGet {
			fmt.Fprint(w, `{"ref":"refs/heads/main","object":{"sha":"abc123"}}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/git/refs" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"ref":"refs/heads/newbranch","object":{"sha":"abc123"}}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "newbranch",
	}

	require.NoError(t, client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message"))
}

func TestGitHubCreateFileGetContentsError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not get")
}

func TestGitHubCreateFileUpdateError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"default_branch": "main"}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"sha": "existing-sha"}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/someone/something/contents/file.txt" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{}, repo, []byte("content"), "file.txt", "message")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not update")
}

func TestGitHubCreateReleaseDeleteDraftError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate:         "v1.0.0",
				Draft:                true,
				ReplaceExistingDraft: true,
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "test release")
	require.Error(t, err)
}

func TestGitHubCreateReleaseTargetCommitishBadTemplate(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Release: config.Release{
				NameTemplate:    "v1.0.0",
				TargetCommitish: "{{ .NoKeyLikeThat }}",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, ctx.Token)
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "test release")
	require.Error(t, err)
	testlib.RequireTemplateError(t, err)
}

func TestGitHubCreateReleaseCreateError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/tags/v1.0.0" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				NameTemplate: "v1.0.0",
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "test release")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not release")
}

func TestGitHubCreateOrUpdateReleaseUpdate(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/tags/v1.0.0" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": 42, "name": "v1.0.0", "body": "old body", "draft": true}`)
			return
		}
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/42" && r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": 42, "name": "v1.0.0", "body": "updated body"}`)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			GitHubURLs: config.GitHubURLs{
				API: srv.URL,
			},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag: "v1.0.0",
		}),
	)

	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	data := &github.RepositoryRelease{
		TagName: new("v1.0.0"),
		Name:    new("v1.0.0"),
		Body:    new("new body"),
		Draft:   new(true),
	}
	release, err := client.createOrUpdateRelease(ctx, data, "new body")
	require.NoError(t, err)
	require.Equal(t, int64(42), release.GetID())
}

func TestGitHubDeleteReleaseArtifactListError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/owner/name/releases/123/assets" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "owner", Name: "name"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteReleaseArtifact(ctx, 123, "artifact.tar.gz", 1)
	require.Error(t, err)
}

func TestGitHubDeleteReleaseArtifactDeleteError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/owner/name/releases/123/assets" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":456,"name":"artifact.tar.gz"}]`)
			return
		}
		if r.URL.Path == "/api/v3/repos/owner/name/releases/assets/456" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "owner", Name: "name"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteReleaseArtifact(ctx, 123, "artifact.tar.gz", 1)
	require.Error(t, err)
}

func TestGitHubDeleteReleaseArtifactNotFound(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/owner/name/releases/123/assets" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":456,"name":"other-file.tar.gz"}]`)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "owner", Name: "name"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteReleaseArtifact(ctx, 123, "artifact.tar.gz", 1)
	require.NoError(t, err)
}

func TestGitHubDeleteReleaseArtifactPagination(t *testing.T) {
	t.Parallel()
	var srvURL string
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/owner/name/releases/123/assets" && r.Method == http.MethodGet {
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/repos/owner/name/releases/123/assets?page=2>; rel="next"`, srvURL))
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"id":456,"name":"other-file.tar.gz"}]`)
				return
			}
			if page == "2" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"id":789,"name":"artifact.tar.gz"}]`)
				return
			}
		}
		if r.URL.Path == "/api/v3/repos/owner/name/releases/assets/789" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	srvURL = srv.URL

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "owner", Name: "name"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteReleaseArtifact(ctx, 123, "artifact.tar.gz", 1)
	require.NoError(t, err)
}

func TestGitHubFindDraftReleaseError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "goreleaser", Name: "test"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.findDraftRelease(ctx, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not list existing drafts")
}

func TestGitHubFindDraftReleasePagination(t *testing.T) {
	t.Parallel()
	var srvURL string
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/repos/goreleaser/test/releases?page=2>; rel="next"`, srvURL))
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"id":10,"name":"v0.9.0","draft":false}]`)
				return
			}
			if page == "2" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"id":20,"name":"v1.0.0","draft":true}]`)
				return
			}
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	srvURL = srv.URL

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "goreleaser", Name: "test"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	release, err := client.findDraftRelease(ctx, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, release)
	require.Equal(t, int64(20), release.GetID())
}

func TestGitHubDeleteExistingDraftReleaseFindError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "goreleaser", Name: "test"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteExistingDraftRelease(ctx, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not delete existing drafts")
}

func TestGitHubDeleteExistingDraftReleaseDeleteError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":1,"name":"v1.0.0","draft":true}]`)
			return
		}
		if r.URL.Path == "/api/v3/repos/goreleaser/test/releases/1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
		Release: config.Release{
			GitHub: config.Repo{Owner: "goreleaser", Name: "test"},
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.deleteExistingDraftRelease(ctx, "v1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not delete previous draft release")
}

func TestGitHubGetMilestoneByTitlePagination(t *testing.T) {
	t.Parallel()
	var srvURL string
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/milestones" && r.Method == http.MethodGet {
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/v3/repos/someone/something/milestones?page=2>; rel="next"`, srvURL))
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"number":1,"title":"v0.9.0"}]`)
				return
			}
			if page == "2" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `[{"number":2,"title":"v1.0.0"}]`)
				return
			}
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})
	srvURL = srv.URL

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	milestone, err := client.getMilestoneByTitle(ctx, repo, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, milestone)
	require.Equal(t, "v1.0.0", milestone.GetTitle())
}

func TestGitHubGetMilestoneByTitleError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v3/repos/someone/something/milestones" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Error("unhandled request: " + r.Method + " " + r.URL.Path)
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	_, err = client.getMilestoneByTitle(ctx, repo, "v1.0.0")
	require.Error(t, err)
}

func TestGitHubUploadParseError(t *testing.T) {
	t.Parallel()
	srv := githubTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
	})

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitHubURLs: config.GitHubURLs{
			API: srv.URL,
		},
	})
	client, err := newGitHub(ctx, "test-token")
	require.NoError(t, err)

	err = client.Upload(ctx, "not-a-number", &artifact.Artifact{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing")
}

// githubTestServer creates a test HTTP server with automatic rate_limit handling
// and cleanup. The provided handler is called for all non-rate-limit requests.
func githubTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120},"search":{"remaining":10}}}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}
