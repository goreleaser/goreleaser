package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
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
	testlib.RequireTemplateError(t, err)
}

func TestGitHubGetDefaultBranch(t *testing.T) {
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
			return
		}

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
	require.Equal(t, 2, totalRequests)
}

func TestGitHubGetDefaultBranchErr(t *testing.T) {
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

func TestGitHubChangelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/compare/v1.0.0...v1.1.0" {
			r, err := os.Open("testdata/github/compare.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}
		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
	require.Equal(t, []ChangelogItem{
		{
			SHA:            "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			Message:        "Fix all the bugs",
			AuthorName:     "Octocat",
			AuthorEmail:    "octo@cat",
			AuthorUsername: "octocat",
		},
	}, log)
}

func TestGitHubReleaseNotes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/releases/generate-notes" {
			r, err := os.Open("testdata/github/releasenotes.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}
		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

func TestGitHubReleaseNotesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/releases/generate-notes" {
			w.WriteHeader(http.StatusBadRequest)
		}
		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

	_, err = client.GenerateReleaseNotes(ctx, repo, "v1.0.0", "v1.1.0")
	require.Error(t, err)
}

func TestGitHubCloseMilestone(t *testing.T) {
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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

const testPRTemplate = "fake template\n- [ ] mark this\n---"

func TestGitHubOpenPullRequestCrossRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			content := github.RepositoryContent{
				Encoding: github.String("base64"),
				Content:  github.String(base64.StdEncoding.EncodeToString([]byte(testPRTemplate))),
			}
			bts, _ := json.Marshal(content)
			_, _ = w.Write(bts)
			return
		}

		if r.URL.Path == "/repos/someone/something/pulls" {
			got, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var pr github.NewPullRequest
			require.NoError(t, json.Unmarshal(got, &pr))
			require.Equal(t, "main", pr.GetBase())
			require.Equal(t, "someoneelse:something:foo", pr.GetHead())
			require.Equal(t, testPRTemplate+"\n"+prFooter, pr.GetBody())
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			content := github.RepositoryContent{
				Encoding: github.String("base64"),
				Content:  github.String(base64.StdEncoding.EncodeToString([]byte(testPRTemplate))),
			}
			bts, _ := json.Marshal(content)
			_, _ = w.Write(bts)
			return
		}

		if r.URL.Path == "/repos/someone/something/pulls" {
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
}

func TestGitHubOpenPullRequestNoBaseBranchDraft(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/repos/someone/something/pulls" {
			got, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var pr github.NewPullRequest
			require.NoError(t, json.Unmarshal(got, &pr))
			require.Equal(t, "main", pr.GetBase())
			require.Equal(t, "someone:something:foo", pr.GetHead())
			require.True(t, pr.GetDraft())

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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{
		Branch: "foo",
	}, "some title", true))
}

func TestGitHubOpenPullRequestPRExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/repos/someone/something/pulls" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			r, err := os.Open("testdata/github/pull.json")
			require.NoError(t, err)
			_, err = io.Copy(w, r)
			require.NoError(t, err)
			return
		}

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
}

func TestGitHubOpenPullRequestBaseEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
		Owner:  "someone",
		Name:   "something",
		Branch: "foo",
	}

	require.NoError(t, client.OpenPullRequest(ctx, Repo{}, repo, "some title", false))
}

func TestGitHubOpenPullRequestHeadEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/repos/someone/something/contents/.github/PULL_REQUEST_TEMPLATE.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, client.OpenPullRequest(ctx, repo, Repo{}, "some title", false))
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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

func TestGitHubCreateFileFeatureBranchAlreadyExists(t *testing.T) {
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
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprintf(w, `{"message": "Reference already exists"}`)
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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"resources":{"core":{"remaining":120}}}`)
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

func TestGitHubCheckRateLimit(t *testing.T) {
	now := time.Now().UTC()
	reset := now.Add(1392 * time.Millisecond)
	var first atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusOK)
			resetstr, _ := github.Timestamp{Time: reset}.MarshalJSON()
			if first.Load() {
				// second time asking for the rate limit
				fmt.Fprintf(w, `{"resources":{"core":{"remaining":138,"reset":%s}}}`, string(resetstr))
				return
			}

			// first time asking for the rate limit
			fmt.Fprintf(w, `{"resources":{"core":{"remaining":98,"reset":%s}}}`, string(resetstr))
			first.Store(true)
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
	client.checkRateLimit(ctx)
	require.True(t, time.Now().UTC().After(reset))
}

// TODO: test create release
// TODO: test create upload file to release
// TODO: test delete draft release
// TODO: test create PR
