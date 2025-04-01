package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

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
		{
			name:            "download_url_string",
			downloadURL:     "https://gitlab.mycompany.com",
			wantDownloadURL: "https://gitlab.mycompany.com/",
		},
	}

	for _, tt := range tests {
		ctx := testctx.NewWithCfg(config.Project{
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

			ctx := testctx.NewWithCfg(config.Project{
				Env:        envs,
				GitLabURLs: gitlabURLs,
			})

			client, err := newGitLab(ctx, ctx.Token)
			require.NoError(t, err)
			require.Equal(t, tt.wantHost, client.client.BaseURL().Host)
		})
	}

	t.Run("no_env_specified", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
			},
		})

		_, err := newGitLab(ctx, ctx.Token)
		require.ErrorAs(t, err, &template.ExecError{})
	})

	t.Run("invalid_template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GitLabURLs: config.GitLabURLs{
				API: "{{.dddddddddd",
			},
		})

		_, err := newGitLab(ctx, ctx.Token)
		require.Error(t, err)
	})
}

func TestGitLabURLsDownloadTemplate(t *testing.T) {
	tests := []struct {
		name               string
		usePackageRegistry bool
		downloadURL        string
		wantURL            string
		wantErr            bool
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
		{
			name:               "url_registry",
			wantURL:            "/api/v4/projects/test%2Ftest/packages/generic/projectname/1%2E0%2E0/test",
			usePackageRegistry: true,
		},
	}

	for _, tt := range tests {
		for _, version := range []string{"16.3.4", "17.1.2"} {
			t.Run(tt.name+"_"+version, func(t *testing.T) {
				first := true
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer r.Body.Close()

					if strings.Contains(r.URL.Path, "version") {
						fmt.Fprintf(w, `{"version":%q}`, version)
						w.WriteHeader(http.StatusOK)
						return
					}

					if !strings.Contains(r.URL.Path, "assets/links") {
						_, _ = io.Copy(io.Discard, r.Body)
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, "{}")
						return
					}

					if first {
						http.Error(w, `{"message":{"name":["has already been taken"]}}`, http.StatusBadRequest)
						first = false
						return
					}

					defer w.WriteHeader(http.StatusOK)
					defer fmt.Fprint(w, "{}")
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err)

					reqBody := map[string]string{}
					assert.NoError(t, json.Unmarshal(b, &reqBody))

					if version[:2] == "17" {
						assert.NotEmpty(t, reqBody["direct_asset_path"])
					} else {
						assert.NotEmpty(t, reqBody["filepath"])
					}

					url := reqBody["url"]
					assert.Truef(t, strings.HasSuffix(url, tt.wantURL), "expected %q to end with %q", url, tt.wantURL)
				}))
				defer srv.Close()

				ctx := testctx.NewWithCfg(config.Project{
					ProjectName: "projectname",
					Env: []string{
						"GORELEASER_TEST_GITLAB_URLS_DOWNLOAD=https://gitlab.mycompany.com",
					},
					Release: config.Release{
						GitLab: config.Repo{
							Owner: "test",
							Name:  "test",
						},
						ReplaceExistingArtifacts: true,
					},
					GitLabURLs: config.GitLabURLs{
						API:                srv.URL,
						Download:           tt.downloadURL,
						UsePackageRegistry: tt.usePackageRegistry,
					},
				}, testctx.WithVersion("1.0.0"))

				tmpFile, err := os.CreateTemp(t.TempDir(), "")
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = tmpFile.Close()
				})

				client, err := newGitLab(ctx, ctx.Token)
				require.NoError(t, err)

				err = client.Upload(ctx, "1234", &artifact.Artifact{Name: "test", Path: "some-path"}, tmpFile)
				if errors.As(err, &RetriableError{}) {
					err = client.Upload(ctx, "1234", &artifact.Artifact{Name: "test", Path: "some-path"}, tmpFile)
				}
				if tt.wantErr {
					require.Error(t, err)
					retriable := errors.As(err, &RetriableError{})
					require.False(t, retriable, "should be a final error")
					return
				}
				require.NoError(t, err)
			})
		}
	}
}

func TestGitLabCreateReleaseUnknownHost(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Release: config.Release{
			GitLab: config.Repo{
				Owner: "owner",
				Name:  "name",
			},
		},
		GitLabURLs: config.GitLabURLs{
			API: "http://goreleaser-notexists",
		},
	})
	client, err := newGitLab(ctx, "test-token")
	require.NoError(t, err)

	_, err = client.CreateRelease(ctx, "body")
	require.Error(t, err)
}

func TestGitLabCreateReleaseReleaseNotExists(t *testing.T) {
	notExistsStatusCodes := []int{http.StatusNotFound, http.StatusForbidden}

	for _, tt := range notExistsStatusCodes {
		t.Run(strconv.Itoa(tt), func(t *testing.T) {
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

			ctx := testctx.NewWithCfg(config.Project{
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

	ctx := testctx.NewWithCfg(config.Project{
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

func TestGitLabCreateReleaseUnknownHTTPError(t *testing.T) {
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.NewWithCfg(config.Project{
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
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/version") {
			return
		}
		t.Error("shouldn't have made any calls to the API")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.NewWithCfg(config.Project{
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
	b, err := client.getDefaultBranch(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, "foo", b)
}

func TestGitLabGetDefaultBranchErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Assume the request to create a branch was good
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/repository/compare") {
			r, err := os.Open("testdata/gitlab/compare.json")
			if assert.NoError(t, err) {
				defer r.Close()
				_, err = io.Copy(w, r)
				assert.NoError(t, err)
			}
			return
		}
		defer r.Body.Close()
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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
			SHA:            "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			Message:        "Fix all the bugs",
			AuthorName:     "Joey User",
			AuthorEmail:    "joey@user.edu",
			AuthorUsername: "",
		},
	}, log)
}

func TestGitLabCreateFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	ctx := testctx.NewWithCfg(config.Project{
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "projects/someone/something/milestones") {
			r, err := os.Open("testdata/gitlab/milestones.json")
			if assert.NoError(t, err) {
				defer r.Close()
				_, err = io.Copy(w, r)
				assert.NoError(t, err)
			}
			return
		} else if strings.HasSuffix(r.URL.Path, "projects/someone/something/milestones/12") {
			r, err := os.Open("testdata/gitlab/milestone.json")
			if assert.NoError(t, err) {
				defer r.Close()
				_, err = io.Copy(w, r)
				assert.NoError(t, err)
			}
			return
		}
		defer r.Body.Close()
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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
			ctx := testctx.NewWithCfg(config.Project{
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
			assert.Equal(t, 32156, pr.TargetProjectID)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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
	require.NoError(t, client.OpenPullRequest(ctx, base, head, "some title", false))
}

func TestGitLabOpenPullRequestBaseEmpty(t *testing.T) {
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
			assert.Equal(t, 0, pr.TargetProjectID)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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

	require.NoError(t, client.OpenPullRequest(ctx, Repo{}, repo, "some title", false))
}

func TestGitLabOpenPullRequestDraft(t *testing.T) {
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
			assert.Equal(t, 0, pr.TargetProjectID)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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

	require.NoError(t, client.OpenPullRequest(ctx, Repo{}, repo, "some title", true))
}

func TestGitLabOpenPullBaseBranchGiven(t *testing.T) {
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
			assert.Equal(t, 0, pr.TargetProjectID)

			_, err = io.Copy(w, strings.NewReader(`{"web_url": "https://gitlab.com/someoneelse/something/merge_requests/1"}`))
			assert.NoError(t, err)
			return
		}

		t.Error("unhandled request: " + r.URL.Path)
	}))
	defer srv.Close()

	ctx := testctx.NewWithCfg(config.Project{
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

	require.NoError(t, client.OpenPullRequest(ctx, Repo{Branch: "main"}, repo, "some title", false))
}
