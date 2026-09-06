package client

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// serveGitLabVersion answers the version probe of newGitLab. Without it the
// probe reads an empty body, which decodes to io.EOF, and is retried.
func serveGitLabVersion(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasSuffix(r.URL.Path, "/version") {
		return false
	}
	fmt.Fprint(w, `{"version":"17.1.2"}`)
	return true
}

func TestGitLabReleaseURLTemplate(t *testing.T) {
	repo := config.Repo{
		Owner: "owner",
		Name:  "name",
	}
	tests := []struct {
		name            string
		repo            config.Repo
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "default_download_url",
			downloadURL:     DefaultGitLabDownloadURL,
			repo:            repo,
			wantDownloadURL: "https://gitlab.com/owner/name/-/releases/{{ urlPathEscape .Tag }}/downloads/{{ .ArtifactName }}",
		},
		{
			name:            "default_download_url_no_owner",
			downloadURL:     DefaultGitLabDownloadURL,
			repo:            config.Repo{Name: "name"},
			wantDownloadURL: "https://gitlab.com/name/-/releases/{{ urlPathEscape .Tag }}/downloads/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			repo:            repo,
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITLAB_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://gitlab.mycompany.com/owner/name/-/releases/{{ urlPathEscape .Tag }}/downloads/{{ .ArtifactName }}",
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
	}

	t.Setenv("CI_SERVER_VERSION", "18.0.0")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Env: []string{
					"GORELEASER_TEST_GITLAB_URLS_DOWNLOAD=https://gitlab.mycompany.com",
				},
				GitLabURLs: config.GitLabURLs{
					Download: tt.downloadURL,
				},
				Release: config.Release{
					GitLab: tt.repo,
				},
			})
			client, err := newGitLab(ctx, ctx.Token)
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

func TestGitLabURLsAPITemplate(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "18.0.0")
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
			name:     "specified_api_env_key",
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

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Env:        envs,
				GitLabURLs: gitlabURLs,
			})

			client, err := newGitLab(ctx, ctx.Token)
			require.NoError(t, err)
			require.Equal(t, tt.wantHost, client.client.BaseURL().Host)
		})
	}

	t.Run("no_env_specified", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitLab(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("invalid_template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := newGitLab(ctx, ctx.Token)
		require.Error(t, err)
	})
}

func TestGitLabURLsDownloadTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		usePackageRegistry bool
		downloadURL        string
		// wantURL is the release link URL the client must send to GitLab. For
		// the package registry it is relative to the API server.
		wantURL   string
		wantErrIs string
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
			wantErrIs:   `map has no entry for key "Eenv"`,
		},
		{
			name:        "download_url_template_invalid",
			downloadURL: "{{.dddddddddd",
			wantErrIs:   `unclosed action`,
		},
		{
			name:        "download_url_string",
			downloadURL: "https://gitlab.mycompany.com",
			wantURL:     "https://gitlab.mycompany.com/",
		},
		{
			name:               "url_registry",
			wantURL:            "/api/v4/projects/test%2Ftest/packages/generic/projectname/1%2E0%2E0/test",
			usePackageRegistry: true,
		},
	}

	for _, version := range []string{"16.3.4", "17.1.2"} {
		for _, tt := range tests {
			t.Run(tt.name+"_"+version, func(t *testing.T) {
				t.Parallel()

				// the release link URL, as GitLab received it.
				var gotURL atomic.Pointer[string]
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()

					if strings.Contains(r.URL.Path, "version") {
						fmt.Fprintf(w, `{"version":%q}`, version)
						return
					}

					if !strings.Contains(r.URL.Path, "assets/links") {
						_, _ = io.Copy(io.Discard, r.Body)
						fmt.Fprint(w, "{}")
						return
					}

					b, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					reqBody := map[string]string{}
					if err := json.Unmarshal(b, &reqBody); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					// GitLab renamed filepath to direct_asset_path in v17.
					pathField := "filepath"
					if strings.HasPrefix(version, "17.") {
						pathField = "direct_asset_path"
					}
					if reqBody[pathField] == "" {
						http.Error(w, "expected "+pathField+" in "+string(b), http.StatusBadRequest)
						return
					}

					url := reqBody["url"]
					gotURL.Store(&url)
					fmt.Fprint(w, "{}")
				}))
				t.Cleanup(srv.Close)

				ctx := testctx.WrapWithCfg(t.Context(), config.Project{
					ProjectName: "projectname",
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
						API:                srv.URL,
						Download:           tt.downloadURL,
						UsePackageRegistry: tt.usePackageRegistry,
					},
				}, testctx.WithVersion("1.0.0"))

				tmpFile, err := os.CreateTemp(t.TempDir(), "")
				require.NoError(t, err)
				_ = tmpFile.Close()

				client, err := newGitLab(ctx, ctx.Token)
				require.NoError(t, err)

				err = client.Upload(ctx, "1234", &artifact.Artifact{Name: "test", Path: tmpFile.Name()})
				if tt.wantErrIs != "" {
					require.ErrorContains(t, err, tt.wantErrIs)
					require.Nil(t, gotURL.Load(), "must not create the release link on a template error")
					return
				}
				require.NoError(t, err)

				want := tt.wantURL
				if tt.usePackageRegistry {
					want = srv.URL + want
				}
				got := gotURL.Load()
				require.NotNil(t, got, "no release link was created")
				require.Equal(t, want, *got)
			})
		}
	}
}

func TestGitLabUploadRetriesAreOurs(t *testing.T) {
	t.Parallel()

	var creates atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)

		switch {
		case strings.Contains(r.URL.Path, "version"):
			fmt.Fprint(w, `{"version":"17.1.2"}`)
		case !strings.Contains(r.URL.Path, "assets/links"):
			fmt.Fprint(w, "{}")
		default:
			creates.Add(1)
			http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitLab: config.Repo{Owner: "test", Name: "test"},
		},
		GitLabURLs: config.GitLabURLs{API: srv.URL},
		Retry:      config.Retry{Attempts: 2},
	}, testctx.WithVersion("1.0.0"))

	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)
	_ = tmpFile.Close()

	client, err := newGitLab(ctx, ctx.Token)
	require.NoError(t, err)

	require.Error(t, client.Upload(ctx, "1234", &artifact.Artifact{
		Name: "test",
		Path: tmpFile.Name(),
	}))
	// the SDK must not retry on its own: 2 attempts configured, 2 requests.
	require.EqualValues(t, 2, creates.Load())
}

func TestGitLabCreateReleaseUnknownHost(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitLab: config.Repo{
				Owner: "owner",
				Name:  "name",
			},
		},
		GitLabURLs: config.GitLabURLs{
			// .invalid never resolves (RFC 2606) and the trailing dot makes it
			// fully qualified, so the resolver answers NXDOMAIN at once instead
			// of walking the search domains. A bare single-label name made CI
			// report a timeout instead, which the SDK's HTTP client treats as
			// retriable, and the test spent 23s in backoff.
			API: "http://goreleaser-notexists.invalid.",
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "body")
	require.Error(t, err)
}

func TestGitLabCreateReleaseReleaseNotExists(t *testing.T) {
	t.Parallel()
	notExistsStatusCodes := []int{http.StatusNotFound, http.StatusForbidden}

	for _, tt := range notExistsStatusCodes {
		t.Run(strconv.Itoa(tt), func(t *testing.T) {
			t.Parallel()
			totalRequests := 0
			createdRelease := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				totalRequests++

				if !strings.Contains(r.URL.Path, "releases") {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "{}")
					return
				}

				// Check if release exists
				if r.Method == http.MethodGet {
					w.WriteHeader(tt)
					fmt.Fprint(w, "{}")
					return
				}

				// Create release if it doesn't exist
				if r.Method == http.MethodPost {
					createdRelease = true
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "{}")
					return
				}

				t.Fatal("should not reach here")
			}))
			defer srv.Close()

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				GitLabURLs: config.GitLabURLs{
					API: srv.URL,
				},
			})
			client, err := newGitLab(ctx, "test-token")
			require.NoError(t, err)

			_, err = client.CreateRelease(ctx, "body")
			require.NoError(t, err)
			require.True(t, createdRelease)
			require.Equal(t, 3, totalRequests)
		})
	}
}

func TestGitLabCreateReleaseReleaseExists(t *testing.T) {
	t.Parallel()
	totalRequests := 0
	createdRelease := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		totalRequests++

		if !strings.Contains(r.URL.Path, "releases") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
			return
		}

		// Check if release exists
		if r.Method == http.MethodGet {
			w.WriteHeader(200)
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{
				"description": "original description",
			}))
			return
		}

		// Update release
		if r.Method == http.MethodPut {
			createdRelease = true
			var resBody map[string]string
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&resBody))
			assert.Equal(t, "original description", resBody["description"])
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
			return
		}

		t.Fatal("should not reach here")
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
		Release: config.Release{
			ReleaseNotesMode: config.ReleaseNotesModeKeepExisting,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "body")
	require.NoError(t, err)
	require.True(t, createdRelease)
	require.Equal(t, 3, totalRequests)
}

func TestGitLabCanRelease(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"developer", http.StatusOK, `{"permissions":{"project_access":{"access_level":30}}}`, ""},
		{"maintainer", http.StatusOK, `{"permissions":{"project_access":{"access_level":40}}}`, ""},
		{"group access", http.StatusOK, `{"permissions":{"group_access":{"access_level":40}}}`, ""},
		{"reporter", http.StatusOK, `{"permissions":{"project_access":{"access_level":20}}}`, "developer or higher"},
		{"permissions absent", http.StatusOK, `{}`, ""},
		{"api error", http.StatusNotFound, `{}`, "could not check release permissions"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				if strings.HasSuffix(r.URL.Path, "/api/v4/version") {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"version":"16.5.0"}`)
					return
				}
				if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/projects/") {
					w.WriteHeader(tt.status)
					fmt.Fprint(w, tt.body)
					return
				}
				t.Errorf("unhandled request: %s %s", r.Method, r.URL.Path)
			}))
			defer srv.Close()

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				GitLabURLs: config.GitLabURLs{API: srv.URL},
				Release: config.Release{
					GitLab: config.Repo{Owner: "someone", Name: "something"},
				},
			})

			client, err := newGitLab(ctx, "test-token")
			require.NoError(t, err)

			err = client.CanRelease(ctx)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestGitLabCreateReleaseUnknownHTTPError(t *testing.T) {
	t.Parallel()
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "body")
	require.Error(t, err)
	require.Equal(t, 2, totalRequests)
}

func TestGitLabGetDefaultBranch(t *testing.T) {
	t.Parallel()
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.getDefaultBranch(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 2, totalRequests)
}

func TestGitLabGetDefaultBranchEnv(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGitLabVersion(w, r) {
			return
		}
		t.Error("shouldn't have made any calls to the API")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	t.Setenv("CI_DEFAULT_BRANCH", "foo")
	t.Setenv("CI_PROJECT_PATH", "someone/something")
	b, err := client.getDefaultBranch(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, "foo", b)
}

func TestGitLabGetDefaultBranchEnvProjectID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGitLabVersion(w, r) {
			return
		}
		t.Error("shouldn't have made any calls to the API")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	t.Setenv("CI_DEFAULT_BRANCH", "main")
	t.Setenv("CI_PROJECT_ID", "123456789")
	b, err := client.getDefaultBranch(ctx, Repo{Name: "123456789"})
	require.NoError(t, err)
	require.Equal(t, "main", b)
}

func TestGitLabCreateFileUsesTargetDefaultBranch(t *testing.T) {
	var gotRef string
	var gotBranch string
	var decodeErr error
	var projectLookups int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case serveGitLabVersion(w, r):
		case r.URL.Path == "/api/v4/projects/target/project":
			projectLookups++
			fmt.Fprint(w, `{"default_branch":"trunk"}`)
		case strings.Contains(r.URL.Path, "/repository/files/test.rb") && r.Method == http.MethodGet:
			gotRef = r.URL.Query().Get("ref")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 File Not Found"}`)
		case strings.Contains(r.URL.Path, "/repository/files/test.rb") && r.Method == http.MethodPost:
			var req map[string]string
			decodeErr = json.NewDecoder(r.Body).Decode(&req)
			gotBranch = req["branch"]
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"file_path":"test.rb","branch":"trunk"}`)
		default:
			http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("CI_DEFAULT_BRANCH", "main")
	t.Setenv("CI_PROJECT_PATH", "source/project")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	err = client.CreateFile(
		ctx,
		config.CommitAuthor{Name: "user", Email: "u@e.com"},
		Repo{Owner: "target", Name: "project"},
		[]byte("content"),
		"test.rb",
		"add test",
	)
	require.NoError(t, err)
	require.NoError(t, decodeErr)
	require.Equal(t, 1, projectLookups)
	require.Equal(t, "trunk", gotRef)
	require.Equal(t, "trunk", gotBranch)
}

func TestGitLabCreateFileExplicitBranchIgnoresCIDefaultBranch(t *testing.T) {
	var gotRef string
	var gotBranch string
	var decodeErr error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case serveGitLabVersion(w, r):
		case strings.HasSuffix(r.URL.Path, "/repository/branches/trunk"):
			fmt.Fprint(w, `{"name":"trunk"}`)
		case strings.Contains(r.URL.Path, "/repository/files/test.rb") && r.Method == http.MethodGet:
			gotRef = r.URL.Query().Get("ref")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 File Not Found"}`)
		case strings.Contains(r.URL.Path, "/repository/files/test.rb") && r.Method == http.MethodPost:
			var req map[string]string
			decodeErr = json.NewDecoder(r.Body).Decode(&req)
			gotBranch = req["branch"]
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"file_path":"test.rb","branch":"trunk"}`)
		default:
			http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("CI_DEFAULT_BRANCH", "main")
	t.Setenv("CI_PROJECT_PATH", "source/project")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	err = client.CreateFile(
		ctx,
		config.CommitAuthor{Name: "user", Email: "u@e.com"},
		Repo{Owner: "target", Name: "project", Branch: "trunk"},
		[]byte("content"),
		"test.rb",
		"add test",
	)
	require.NoError(t, err)
	require.NoError(t, decodeErr)
	require.Equal(t, "trunk", gotRef)
	require.Equal(t, "trunk", gotBranch)
}

func TestGitLabGetDefaultBranchErr(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if serveGitLabVersion(w, r) {
			return
		}

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.getDefaultBranch(ctx, repo)
	require.Error(t, err)
}

func TestGitLabChangelog(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGitLabVersion(w, r) {
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/compare") {
			serveTestFile(t, w, "testdata/gitlab/compare.json")
			return
		}
		defer r.Body.Close()
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
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
				Name:     "Joey User",
				Email:    "joey@user.edu",
				Username: "",
			}},
			AuthorName:     "Joey User",
			AuthorEmail:    "joey@user.edu",
			AuthorUsername: "",
		},
	}, log)
}

func TestGitLabCreateFile(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGitLabVersion(w, r) {
			return
		}
		// Handle the test where we know the branch and it exists
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/branches/somebranch") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/files/newfile.txt") {
			_, err := io.Copy(w, strings.NewReader(`{ "file_path": "newfile.txt", "branch": "somebranch" }`))
			assert.NoError(t, err)
			return
		}

		// Handle the test where we detect the branch
		if strings.HasSuffix(r.URL.Path, "projects/someone/something") {
			_, err := io.Copy(w, strings.NewReader(`{ "default_branch": "main" }`))
			assert.NoError(t, err)
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/files/newfile-in-default.txt") {
			_, err := io.Copy(w, strings.NewReader(`{ "file_path": "newfile.txt", "branch": "main" }`))
			assert.NoError(t, err)
			return
		}

		// Handle the test where the branch doesn't exist already
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/branches/non-existing-branch") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/files/newfile-on-new-branch.txt") {
			if r.Method == "POST" {
				var resBody map[string]string
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&resBody))
				assert.Equal(t, "master", resBody["start_branch"])
			}
			_, err := io.Copy(w, strings.NewReader(`{"file_path":"newfile-on-new-branch.txt","branch":"non-existing-branch"}`))
			assert.NoError(t, err)
			return
		}

		// Handle the case with a projectID
		if strings.HasSuffix(r.URL.Path, "projects/123456789/repository/branches/main") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/123456789/repository/files/newfile-projectID.txt") {
			_, err := io.Copy(w, strings.NewReader(`{ "file_path": "newfile-projectID.txt", "branch": "main" }`))
			assert.NoError(t, err)
			return
		}
		// File of doooom...gets created, but 404s when getting fetched
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/files/doomed-file-404.txt") {
			if r.Method == "PUT" {
				_, err := io.Copy(w, strings.NewReader(`{ "file_path": "doomed-file-404.txt", "branch": "main" }`))
				assert.NoError(t, err)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		defer r.Body.Close()
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})

	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	// Test using an arbitrary existing branch
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{Name: repo.Owner}, repo, []byte("Hello there"), "newfile.txt", "test: test commit")
	require.NoError(t, err)

	// Test detecting the default branch
	repo = Repo{
		Owner: "someone",
		Name:  "something",
		// Note there is no branch here, gonna try and guess it!
	}

	err = client.CreateFile(ctx, config.CommitAuthor{Name: repo.Owner}, repo, []byte("Hello there"), "newfile-in-default.txt", "test: test commit")
	require.NoError(t, err)

	// Test creating a new branch
	repo = Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "non-existing-branch",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{Name: repo.Owner}, repo, []byte("Hello there"), "newfile-on-new-branch.txt", "test: test commit")
	require.NoError(t, err)

	// Test using projectID
	repo = Repo{
		Name:   "123456789",
		Branch: "main",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{Name: repo.Owner}, repo, []byte("Hello there"), "newfile-projectID.txt", "test: test commit")
	require.NoError(t, err)

	// Test a doomed file. This is a file that is 'successfully' created, but returns a 404 when trying to fetch
	repo = Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "doomed",
	}

	err = client.CreateFile(ctx, config.CommitAuthor{Name: repo.Owner}, repo, []byte("Hello there"), "doomed-file-404.txt", "test: test commit")
	require.Error(t, err)
}

func TestGitLabCloseMilestone(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGitLabVersion(w, r) {
			return
		}
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/milestones") {
			serveTestFile(t, w, "testdata/gitlab/milestones.json")
			return
		} else if strings.HasSuffix(r.URL.Path, "projects/someone/something/milestones/12") {
			serveTestFile(t, w, "testdata/gitlab/milestone.json")
			return
		}
		defer r.Body.Close()
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	repo := Repo{
		Owner: "someone",
		Name:  "something",
	}

	err = client.CloseMilestone(ctx, repo, "10.0")
	require.NoError(t, err)

	// Be sure to error on missing milestones
	err = client.CloseMilestone(ctx, repo, "never-will-exist")
	require.Error(t, err)
}

func TestGitLabCheckUseJobToken(t *testing.T) {
	tests := []struct {
		useJobToken bool
		token       string
		ciToken     string
		want        bool
		desc        string
		name        string
	}{
		{
			useJobToken: true,
			token:       "real-ci-token",
			ciToken:     "real-ci-token",
			desc:        "token and CI_JOB_TOKEN match so should return true",
			want:        true,
			name:        "UseJobToken-tokens-equal",
		},
		{
			useJobToken: true,
			token:       "some-random-token",
			ciToken:     "real-ci-token",
			desc:        "token and CI_JOB_TOKEN do NOT match so should return false",
			want:        false,
			name:        "UseJobToken-tokens-diff",
		},
		{
			useJobToken: false,
			token:       "real-ci-token",
			ciToken:     "real-ci-token",
			desc:        "token and CI_JOB_TOKEN match, however UseJobToken is set to false, so return false",
			want:        false,
			name:        "NoUseJobToken-tokens-equal",
		},
		{
			useJobToken: false,
			token:       "real-ci-token",
			ciToken:     "real-ci-token",
			desc:        "token and CI_JOB_TOKEN do not match, and UseJobToken is set to false, should return false",
			want:        false,
			name:        "NoUseJobToken-tokens-diff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI_JOB_TOKEN", tt.ciToken)
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				GitLabURLs: config.GitLabURLs{
					UseJobToken: tt.useJobToken,
				},
			})
			got := checkUseJobToken(*ctx, tt.token)
			require.Equal(t, tt.want, got, tt.desc)
		})
	}
}

func TestGitLabOpenPullRequestCrossRepo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v4/version" {
			_, err := io.Copy(w, strings.NewReader(`{ "version": "17.1.2" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something" {
			_, err := io.Copy(w, strings.NewReader(`{ "id": 32156 }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someoneelse/something/merge_requests" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr gitlab.MergeRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.TargetBranch)
			assert.Equal(t, "foo", pr.SourceBranch)
			assert.Equal(t, "some title", pr.Title)
			assert.EqualValues(t, 32156, pr.TargetProjectID)
			assert.Equal(t, prFooter, pr.Description)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})

	client, err := newGitLab(ctx, "test-token")
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
	require.NoError(t, prErr(client.OpenPullRequest(ctx, base, head, "some title", false)))
}

func TestGitLabOpenPullRequestBaseEmpty(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v4/version" {
			_, err := io.Copy(w, strings.NewReader(`{ "version": "17.1.2" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something" {
			_, err := io.Copy(w, strings.NewReader(`{ "default_branch": "main" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something/merge_requests" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr gitlab.MergeRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.TargetBranch)
			assert.Equal(t, "foo", pr.SourceBranch)
			assert.Equal(t, "some title", pr.Title)
			assert.EqualValues(t, 0, pr.TargetProjectID)
			assert.Equal(t, prFooter, pr.Description)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})

	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "foo",
	}

	require.NoError(t, prErr(client.OpenPullRequest(ctx, Repo{}, repo, "some title", false)))
}

func TestGitLabOpenPullRequestDraft(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v4/version" {
			_, err := io.Copy(w, strings.NewReader(`{ "version": "17.1.2" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something" {
			_, err := io.Copy(w, strings.NewReader(`{ "default_branch": "main" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something/merge_requests" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr gitlab.MergeRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.TargetBranch)
			assert.Equal(t, "main", pr.SourceBranch)
			assert.Equal(t, "Draft: some title", pr.Title)
			assert.EqualValues(t, 0, pr.TargetProjectID)
			assert.Equal(t, prFooter, pr.Description)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})

	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "main",
	}

	require.NoError(t, prErr(client.OpenPullRequest(ctx, Repo{}, repo, "some title", true)))
}

func TestGitLabOpenPullRequestBaseBranchGiven(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.URL.Path == "/api/v4/version" {
			_, err := io.Copy(w, strings.NewReader(`{ "version": "17.1.2" }`))
			assert.NoError(t, err)
			return
		}

		if r.URL.Path == "/api/v4/projects/someone/something/merge_requests" {
			got, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var pr gitlab.MergeRequest
			assert.NoError(t, json.Unmarshal(got, &pr))
			assert.Equal(t, "main", pr.TargetBranch)
			assert.Equal(t, "foo", pr.SourceBranch)
			assert.Equal(t, "some title", pr.Title)
			assert.EqualValues(t, 0, pr.TargetProjectID)
			assert.Equal(t, prFooter, pr.Description)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{
			API: srv.URL,
		},
	})

	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "foo",
	}

	require.NoError(t, prErr(client.OpenPullRequest(ctx, Repo{Branch: "main"}, repo, "some title", false)))
}

func TestGitLabVersionEnv(t *testing.T) {
	t.Run("18", func(t *testing.T) {
		t.Setenv("CI_SERVER_VERSION", "18.0.0")
		require.True(t, isV17(testctx.Wrap(t.Context()), nil))
	})
	t.Run("17", func(t *testing.T) {
		t.Setenv("CI_SERVER_VERSION", "17.0.0")
		require.True(t, isV17(testctx.Wrap(t.Context()), nil))
	})
	t.Run("16", func(t *testing.T) {
		t.Setenv("CI_SERVER_VERSION", "16.0.0")
		require.False(t, isV17(testctx.Wrap(t.Context()), nil))
	})
}

func TestGitLabVersionProbeIsBounded(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "")

	var probes atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		probes.Add(1)
		http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	// the release retry budget is deliberately huge: if the probe used it,
	// goreleaser would look wedged for ~25 minutes.
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
		Retry: config.Retry{
			Attempts: 10,
			Delay:    10 * time.Second,
			MaxDelay: 5 * time.Minute,
		},
	})

	start := time.Now()
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	require.False(t, client.isV17OrLater)
	require.EqualValues(t, versionRetry.Attempts, probes.Load())
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestGitLabVersionRequestUsesReleaseContext(t *testing.T) {
	entered := make(chan struct{}, 1)
	releaseResponse := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !strings.HasPrefix(r.URL.Path, "/api/v4/version") {
			http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		select {
		case entered <- struct{}{}:
		default:
		}
		<-releaseResponse
		fmt.Fprint(w, `{"version":"18.0.0"}`)
	}))
	t.Cleanup(srv.Close)

	baseCtx, cancel := stdctx.WithCancel(t.Context())
	ctx := testctx.WrapWithCfg(baseCtx, config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
	})

	done := make(chan error, 1)
	go func() {
		_, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		close(releaseResponse)
		t.Fatal("timed out waiting for GitLab version request")
	}

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		close(releaseResponse)
		<-done
		t.Fatal("GitLab version request did not return after cancellation")
	}
	close(releaseResponse)
}

func TestGitLabChangelogRequestUsesReleaseContext(t *testing.T) {
	entered := make(chan struct{}, 1)
	releaseResponse := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case serveGitLabVersion(w, r):
		case strings.Contains(r.URL.Path, "/repository/compare"):
			select {
			case entered <- struct{}{}:
			default:
			}
			<-releaseResponse
			fmt.Fprint(w, `{"commits":[]}`)
		default:
			http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	baseCtx, cancel := stdctx.WithCancel(t.Context())
	ctx := testctx.WrapWithCfg(baseCtx, config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := client.Changelog(ctx, Repo{Owner: "someone", Name: "something"}, "v1.0.0", "v1.1.0")
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		close(releaseResponse)
		t.Fatal("timed out waiting for GitLab compare request")
	}

	cancel()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		close(releaseResponse)
		<-done
		t.Fatal("GitLab compare request did not return after cancellation")
	}
	close(releaseResponse)
}

func TestGitLabRateLimitRetryAfter(t *testing.T) {
	t.Parallel()
	reset := time.Now().Add(42 * time.Second)
	for name, tt := range map[string]struct {
		header http.Header
		want   time.Duration
		// resetAt, when set, expects the time remaining until it, measured
		// when the subtest runs. A constant goes stale here: the header
		// carries an absolute instant, and t.Parallel() defers the subtest
		// by however long the scheduler takes.
		resetAt time.Time
	}{
		"no headers": {
			header: http.Header{},
		},
		"reset in the future": {
			header: http.Header{
				"Ratelimit-Reset": {strconv.FormatInt(reset.Unix(), 10)},
			},
			resetAt: reset,
		},
		"reset in the past falls back to retry-after": {
			header: http.Header{
				"Ratelimit-Reset": {strconv.FormatInt(reset.Add(-time.Hour).Unix(), 10)},
				"Retry-After":     {"30"},
			},
			want: 30 * time.Second,
		},
		"retry-after only": {
			header: http.Header{"Retry-After": {"7"}},
			want:   7 * time.Second,
		},
		// GitLab always answers with delta-seconds. An HTTP-date is valid per
		// RFC 9110, but unsupported: the retry layer then backs off on its own.
		"retry-after as an http date": {
			header: http.Header{"Retry-After": {"Wed, 21 Oct 2015 07:28:00 GMT"}},
		},
		"garbage": {
			header: http.Header{
				"Ratelimit-Reset": {"nope"},
				"Retry-After":     {"-1"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			want := tt.want
			if !tt.resetAt.IsZero() {
				want = time.Until(tt.resetAt)
			}
			got := rateLimitRetryAfter(tt.header)
			// Unix() truncated the header to whole seconds.
			require.InDelta(t, want, got, float64(time.Second))
		})
	}
}

func TestGitLabErrorRateLimit(t *testing.T) {
	t.Parallel()
	newResp := func(status int, header http.Header) *gitlab.Response {
		return &gitlab.Response{Response: &http.Response{
			StatusCode: status,
			Header:     header,
		}}
	}
	rateLimited := http.Header{"Retry-After": {"30"}}

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, gitlabError(nil, newResp(500, nil)))
	})

	t.Run("429 carries the retry-after", func(t *testing.T) {
		t.Parallel()
		err := gitlabError(errors.New("slow down"), newResp(http.StatusTooManyRequests, rateLimited))
		he, ok := errors.AsType[retryx.HTTPError](err)
		require.True(t, ok)
		require.Equal(t, 30*time.Second, he.RetryAfter)
		require.True(t, retryx.IsRetriable(err))
	})

	t.Run("other statuses ignore the headers", func(t *testing.T) {
		t.Parallel()
		err := gitlabError(errors.New("bad request"), newResp(http.StatusBadRequest, rateLimited))
		he, ok := errors.AsType[retryx.HTTPError](err)
		require.True(t, ok)
		require.Zero(t, he.RetryAfter)
		require.False(t, retryx.IsRetriable(err))
	})

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()
		err := gitlabError(errors.New("boom"), nil)
		he, ok := errors.AsType[retryx.HTTPError](err)
		require.True(t, ok)
		require.Zero(t, he.Status)
	})
}

func TestGitLabCreateFileNewFile(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprint(w, `{"name":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 File Not Found"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"file_path":"test.rb","branch":"main"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.NoError(t, err)
}

func TestGitLabCreateFileUpdateExisting(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprint(w, `{"name":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"file_name":"test.rb","file_path":"test.rb","size":7,"encoding":"base64","content":"Y29udGVudA==","ref":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodPut:
			fmt.Fprint(w, `{"file_path":"test.rb","branch":"main"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "main"}, []byte("updated"), "test.rb", "update test")
	require.NoError(t, err)
}

func TestGitLabCreateFileNewBranch(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/new-branch"):
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 Branch Not Found"}`)
		case strings.HasSuffix(r.URL.Path, "projects/someone/something"):
			fmt.Fprint(w, `{"id":1,"default_branch":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 File Not Found"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"file_path":"test.rb","branch":"new-branch"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "new-branch"}, []byte("content"), "test.rb", "add test")
	require.NoError(t, err)
}

func TestGitLabCreateFileGetFileError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprint(w, `{"name":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.Error(t, err)
}

func TestGitLabCreateFileCreateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprint(w, `{"name":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 File Not Found"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.Error(t, err)
}

func TestGitLabCreateFileUpdateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprint(w, `{"name":"main"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"file_name":"test.rb","file_path":"test.rb"}`)
		case strings.Contains(r.URL.Path, "/repository/files/") && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "someone", Name: "something", Branch: "main"}, []byte("updated"), "test.rb", "update test")
	require.Error(t, err)
}

func TestGitLabOpenPullRequestGetProjectError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "projects/someone/something") && !strings.Contains(r.URL.Path, "merge_requests"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	_, err = client.OpenPullRequest(ctx, Repo{Owner: "someone", Name: "something", Branch: "main"}, Repo{Owner: "someoneelse", Name: "something", Branch: "feature"}, "test PR", false)
	require.Error(t, err)
}

func TestGitLabOpenPullRequestDefaultBranchError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "projects/someone/something"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	_, err = client.OpenPullRequest(ctx, Repo{}, Repo{Owner: "someone", Name: "something", Branch: "feature"}, "test PR", false)
	require.Error(t, err)
}

func TestGitLabOpenPullRequestCreateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "projects/someone/something") && !strings.Contains(r.URL.Path, "merge_requests"):
			fmt.Fprint(w, `{"id":123,"default_branch":"main"}`)
		case strings.Contains(r.URL.Path, "merge_requests"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	_, err = client.OpenPullRequest(ctx, Repo{Owner: "someone", Name: "something", Branch: "main"}, Repo{Owner: "someone", Name: "something", Branch: "feature"}, "test PR", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not create pull request")
}

func TestGitLabReleaseURLTemplateNameError(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "18.0.0")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Release: config.Release{GitLab: config.Repo{Name: "{{ .NoKeyLikeThat }}"}}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	_, err = client.ReleaseURLTemplate(ctx)
	require.Error(t, err)
}

func TestGitLabUploadNameTemplateError(t *testing.T) {
	t.Setenv("CI_SERVER_VERSION", "18.0.0")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Release: config.Release{GitLab: config.Repo{Name: "{{ .NoKeyLikeThat }}"}}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	err = client.Upload(ctx, "v1.0.0", &artifact.Artifact{Name: "test.tar.gz", Path: "testdata/gitlab/milestone.json"})
	require.Error(t, err)
}

func TestGitLabUploadPackageRegistryError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myproject",
		GitLabURLs:  config.GitLabURLs{API: srv.URL, UsePackageRegistry: true},
		Release:     config.Release{GitLab: config.Repo{Owner: "someone", Name: "something"}},
	}, testctx.WithVersion("1.0.0"))
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	err = client.Upload(ctx, "v1.0.0", &artifact.Artifact{Name: "test.tar.gz", Path: "testdata/gitlab/milestone.json"})
	require.Error(t, err)
}

func TestGitLabUploadMarkdownError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
		Release:    config.Release{GitLab: config.Repo{Owner: "someone", Name: "something"}},
	})
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	err = client.Upload(ctx, "v1.0.0", &artifact.Artifact{Name: "test.tar.gz", Path: "testdata/gitlab/milestone.json"})
	require.Error(t, err)
}

func TestGitLabUploadPackageRegistry(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.Contains(r.URL.Path, "packages/generic") && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"message":"201 Created"}`)
		case strings.Contains(r.URL.Path, "assets/links") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":1,"name":"test.tar.gz","direct_asset_url":"http://example.com/test.tar.gz"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myproject",
		GitLabURLs:  config.GitLabURLs{API: srv.URL, UsePackageRegistry: true},
		Release:     config.Release{GitLab: config.Repo{Owner: "someone", Name: "something"}},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	err = client.Upload(ctx, "v1.0.0", &artifact.Artifact{Name: "test.tar.gz", Path: "testdata/gitlab/milestone.json"})
	require.NoError(t, err)
}

func TestGitLabCreateReleaseCreateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.Contains(r.URL.Path, "/releases") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"404 Not Found"}`)
		case strings.Contains(r.URL.Path, "/releases") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
		Release:    config.Release{GitLab: config.Repo{Owner: "someone", Name: "something"}},
	}, testctx.WithCurrentTag("v1.0.0"), testctx.WithCommit("abc123"))
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	_, err = client.CreateRelease(ctx, "release body")
	require.Error(t, err)
}

func TestGitLabCreateReleaseUpdateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.Contains(r.URL.Path, "/releases") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"tag_name":"v1.0.0","name":"Release","description":"old body"}`)
		case strings.Contains(r.URL.Path, "/releases") && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GitLabURLs: config.GitLabURLs{API: srv.URL},
		Release:    config.Release{GitLab: config.Repo{Owner: "someone", Name: "something"}},
	}, testctx.WithCurrentTag("v1.0.0"))
	client, err := newGitLab(ctx, "test-token", gitlab.WithoutRetries())
	require.NoError(t, err)
	_, err = client.CreateRelease(ctx, "new body")
	require.Error(t, err)
}

func TestGitLabGetMilestoneByTitlePagination(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
			fmt.Fprint(w, `{"version":"18.0.0"}`)
		case strings.HasSuffix(r.URL.Path, "milestones"):
			page := r.URL.Query().Get("page")
			switch page {
			case "", "1":
				w.Header().Set("X-Next-Page", "2")
				w.Header().Set("X-Page", "1")
				fmt.Fprint(w, `[{"id":1,"title":"v0.9.0"}]`)
			case "2":
				w.Header().Set("X-Page", "2")
				fmt.Fprint(w, `[{"id":2,"title":"v1.0.0"}]`)
			}
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GitLabURLs: config.GitLabURLs{API: srv.URL}})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)
	milestone, err := client.getMilestoneByTitle(ctx, Repo{Owner: "someone", Name: "something"}, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, milestone)
	require.Equal(t, "v1.0.0", milestone.Title)
}

func TestGitLabPublishRelease(t *testing.T) {
	t.Parallel()
	client := &gitlabClient{}
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, client.PublishRelease(ctx, "123"))
}

func TestGitLabUploadReleaseLinkExists(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		links           string
		linksStatus     int
		replace         bool
		failFirstCreate bool
		retryAttempts   uint
		wantCreates     int64
		wantDeletes     int64
		wantErrs        []string
	}{
		"replaces it without another retry": {
			links:         `[{"id":1,"name":"other"},{"id":2,"name":"test.tar.gz"}]`,
			replace:       true,
			retryAttempts: 1,
			wantCreates:   2,
			wantDeletes:   1,
		},
		"replaces it after an earlier transient failure": {
			links:           `[{"id":1,"name":"other"},{"id":2,"name":"test.tar.gz"}]`,
			replace:         true,
			failFirstCreate: true,
			retryAttempts:   2,
			wantCreates:     3,
			wantDeletes:     1,
		},
		"replace disabled": {
			links:         `[{"id":2,"name":"test.tar.gz"}]`,
			retryAttempts: 2,
			wantCreates:   1,
			wantErrs:      []string{"has already been taken"},
		},
		"no link with that name": {
			links:         `[{"id":1,"name":"other"}]`,
			replace:       true,
			retryAttempts: 2,
			wantCreates:   1,
			wantErrs:      []string{"has already been taken"},
		},
		"listing the links fails": {
			links:         `{"message":"404 Project Not Found"}`,
			linksStatus:   http.StatusNotFound,
			replace:       true,
			retryAttempts: 2,
			wantCreates:   1,
			// the create error must survive: it names the real problem.
			wantErrs: []string{"has already been taken", "404"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var deletes atomic.Int64
			var createAttempts atomic.Int64
			var duplicateReturned atomic.Bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()
				_, _ = io.Copy(io.Discard, r.Body)
				switch {
				case strings.HasPrefix(r.URL.Path, "/api/v4/version"):
					fmt.Fprint(w, `{"version":"18.0.0"}`)
				case strings.Contains(r.URL.Path, "uploads") && r.Method == http.MethodPost:
					fmt.Fprint(w, `{"alt":"test","url":"/uploads/abc/test.tar.gz","full_path":"someone/something/uploads/abc/test.tar.gz","markdown":"[test](/uploads/abc/test.tar.gz)"}`)
				case !strings.Contains(r.URL.Path, "assets/links"):
					fmt.Fprint(w, "{}")
				case r.Method == http.MethodGet:
					if tt.linksStatus != 0 {
						http.Error(w, tt.links, tt.linksStatus)
						return
					}
					fmt.Fprint(w, tt.links)
				case r.Method == http.MethodDelete:
					deletes.Add(1)
					fmt.Fprint(w, `{"id":2,"name":"test.tar.gz"}`)
				case tt.failFirstCreate && createAttempts.Load() == 0:
					createAttempts.Add(1)
					http.Error(w, `{"error":"service unavailable"}`, http.StatusServiceUnavailable)
				case !duplicateReturned.Swap(true):
					// the link already exists, so the first create fails.
					createAttempts.Add(1)
					http.Error(
						w,
						`{"message":{"name":["has already been taken"]}}`,
						http.StatusBadRequest,
					)
				default:
					createAttempts.Add(1)
					fmt.Fprint(w, `{"id":3,"name":"test.tar.gz"}`)
				}
			}))
			t.Cleanup(srv.Close)

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				GitLabURLs: config.GitLabURLs{
					API:      srv.URL,
					Download: srv.URL,
				},
				Release: config.Release{
					GitLab: config.Repo{
						Owner: "someone",
						Name:  "something",
					},
					ReplaceExistingArtifacts: tt.replace,
				},
				Retry: config.Retry{Attempts: tt.retryAttempts},
			}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))
			client, err := newGitLab(ctx, "test-token")
			require.NoError(t, err)

			a := &artifact.Artifact{Name: "test.tar.gz", Path: "testdata/gitlab/milestone.json"}
			err = client.Upload(ctx, "v1.0.0", a)
			if tt.wantErrs == nil {
				require.NoError(t, err)
			}
			for _, want := range tt.wantErrs {
				require.ErrorContains(t, err, want)
			}
			require.Equal(t, tt.wantCreates, createAttempts.Load())
			require.Equal(t, tt.wantDeletes, deletes.Load())
		})
	}
}
