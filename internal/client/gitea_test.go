package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GetInstanceURLSuite struct {
	suite.Suite
}

func (s *GetInstanceURLSuite) TestWithScheme() {
	t := s.T()
	rootURL := "https://gitea.com"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: rootURL + "/api/v1",
		},
	})

	result, err := getInstanceURL(ctx)
	require.NoError(t, err)
	require.Equal(t, rootURL, result)
}

func (s *GetInstanceURLSuite) TestParseError() {
	t := s.T()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "://wrong.gitea.com",
		},
	})

	result, err := getInstanceURL(ctx)
	require.Error(t, err)
	require.Empty(t, result)
}

func (s *GetInstanceURLSuite) TestNoScheme() {
	t := s.T()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "gitea.com",
		},
	})

	result, err := getInstanceURL(ctx)
	require.Error(t, err)
	require.Empty(t, result)
}

func (s *GetInstanceURLSuite) TestEmpty() {
	t := s.T()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "",
		},
	})

	result, err := getInstanceURL(ctx)
	require.Error(t, err)
	require.Empty(t, result)
}

func (s *GetInstanceURLSuite) TestTemplate() {
	t := s.T()
	rootURL := "https://gitea.mycompany.com"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Env: []string{
			fmt.Sprintf("GORELEASER_TEST_GITAEA_URLS_API=%s", rootURL),
		},
		GiteaURLs: config.GiteaURLs{
			API: "{{ .Env.GORELEASER_TEST_GITAEA_URLS_API }}",
		},
	})

	result, err := getInstanceURL(ctx)
	require.NoError(t, err)
	require.Equal(t, rootURL, result)
}

func (s *GetInstanceURLSuite) TestTemplateMissingValue() {
	t := s.T()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
		},
	})

	result, err := getInstanceURL(ctx)
	require.ErrorAs(t, err, &tmpl.Error{})
	require.Empty(t, result)
}

func (s *GetInstanceURLSuite) TestTemplateInvalid() {
	t := s.T()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "{{.dddddddddd",
		},
	})

	result, err := getInstanceURL(ctx)
	require.Error(t, err)
	require.Empty(t, result)
}

func TestGetInstanceURLSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GetInstanceURLSuite))
}

type GiteaReleasesTestSuite struct {
	suite.Suite
	url          string
	owner        string
	repoName     string
	tag          string
	client       *giteaClient
	releasesURL  string
	title        string
	description  string
	ctx          *context.Context
	commit       string
	isDraft      bool
	isPrerelease bool
	releaseURL   string
	releaseID    int64
}

func (s *GiteaReleasesTestSuite) SetupTest() {
	httpmock.Activate()
	s.url = "https://gitea.example.com"
	s.owner = "owner"
	s.repoName = "repoName"
	s.releasesURL = fmt.Sprintf(
		"%v/api/v1/repos/%v/%v/releases",
		s.url,
		s.owner,
		s.repoName,
	)
	s.tag = "tag"
	s.title = "gitea_release_title"
	s.description = "gitea release description"
	s.commit = "some commit hash"
	s.isDraft = false
	s.isPrerelease = true
	s.ctx = testctx.WrapWithCfg(
		s.T().Context(),
		config.Project{
			ProjectName: "project",
			Release: config.Release{
				NameTemplate: "{{ .ProjectName }}_{{ .Version }}",
				Gitea: config.Repo{
					Owner: s.owner,
					Name:  s.repoName,
				},
				Draft: s.isDraft,
			},
		},
		testctx.WithVersion("6.6.6"),
		testctx.WithGitInfo(context.GitInfo{
			CurrentTag:  s.tag,
			Commit:      s.commit,
			ShortCommit: s.commit[0:2],
			URL:         "https://gitea.com/goreleaser/goreleaser.git",
		}),
		func(ctx *context.Context) {
			ctx.PreRelease = s.isPrerelease
		},
		testctx.WithSemver(6, 6, 6, ""),
	)

	s.releaseID = 666
	s.releaseURL = fmt.Sprintf("%v/%v", s.releasesURL, s.releaseID)
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/version", s.url), httpmock.NewStringResponder(200, "{\"version\":\"1.12.0\"}"))
	newClient, err := gitea.NewClient(s.url)
	s.Require().NoError(err)
	s.client = &giteaClient{client: newClient}
}

func (s *GiteaReleasesTestSuite) TearDownTest() {
	httpmock.DeactivateAndReset()
}

type GetExistingReleaseSuite struct {
	GiteaReleasesTestSuite
}

func (s *GetExistingReleaseSuite) TestNoReleases() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(200, "[]"))

	release, err := s.client.getExistingRelease(s.ctx, s.owner, s.repoName, s.tag)
	require.Nil(t, release)
	require.NoError(t, err)
}

func (s *GetExistingReleaseSuite) TestNoRepo() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(404, ""))

	release, err := s.client.getExistingRelease(s.ctx, s.owner, s.repoName, s.tag)
	require.Nil(t, release)
	require.Error(t, err)
}

func (s *GetExistingReleaseSuite) TestReleaseExists() {
	t := s.T()
	release := gitea.Release{TagName: s.tag}
	resp, err := httpmock.NewJsonResponder(200, []gitea.Release{release})
	require.NoError(t, err)
	httpmock.RegisterResponder("GET", s.releasesURL, resp)

	result, err := s.client.getExistingRelease(s.ctx, s.owner, s.repoName, s.tag)
	require.NotNil(t, result)
	require.Equal(t, *result, release)
	require.NoError(t, err)
}

func TestGiteaGetExistingReleaseSuite(t *testing.T) {
	suite.Run(t, new(GetExistingReleaseSuite))
}

type GiteacreateReleaseSuite struct {
	GiteaReleasesTestSuite
}

func (s *GiteacreateReleaseSuite) TestSuccess() {
	t := s.T()
	expectedRelease := gitea.Release{
		TagName:      s.tag,
		Target:       s.commit,
		Note:         s.description,
		IsDraft:      s.isDraft,
		IsPrerelease: s.isPrerelease,
	}
	resp, err := httpmock.NewJsonResponder(200, &expectedRelease)
	require.NoError(t, err)
	httpmock.RegisterResponder("POST", s.releasesURL, resp)

	release, err := s.client.createRelease(s.ctx, s.title, s.description)
	require.NoError(t, err)
	require.NotNil(t, release)
	require.Equal(t, expectedRelease, *release)
}

func (s *GiteacreateReleaseSuite) TestError() {
	t := s.T()
	httpmock.RegisterResponder("POST", s.releasesURL, httpmock.NewStringResponder(400, ""))

	release, err := s.client.createRelease(s.ctx, s.title, s.description)
	require.Error(t, err)
	require.Nil(t, release)
}

func TestGiteacreateReleaseSuite(t *testing.T) {
	suite.Run(t, new(GiteacreateReleaseSuite))
}

type GiteaupdateReleaseSuite struct {
	GiteaReleasesTestSuite
}

func (s *GiteaupdateReleaseSuite) TestSuccess() {
	t := s.T()
	expectedRelease := gitea.Release{
		TagName:      s.tag,
		Target:       s.commit,
		Note:         s.description,
		IsDraft:      s.isDraft,
		IsPrerelease: s.isPrerelease,
	}
	resp, err := httpmock.NewJsonResponder(200, &expectedRelease)
	require.NoError(t, err)
	httpmock.RegisterResponder("PATCH", s.releaseURL, resp)

	release, err := s.client.updateRelease(s.ctx, s.title, s.description, s.releaseID)
	require.NoError(t, err)
	require.NotNil(t, release)
}

func (s *GiteaupdateReleaseSuite) TestError() {
	t := s.T()
	httpmock.RegisterResponder("PATCH", s.releaseURL, httpmock.NewStringResponder(400, ""))

	release, err := s.client.updateRelease(s.ctx, s.title, s.description, s.releaseID)
	require.Error(t, err)
	require.Nil(t, release)
}

func (s *GiteaupdateReleaseSuite) TestGiteaCreateFile() {
	t := s.T()
	fileEndpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s", s.url, s.owner, s.repoName, "file.txt")
	projectEndpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s", s.url, s.owner, s.repoName)

	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/version", s.url), httpmock.NewStringResponder(200, "{\"version\":\"1.12.0\"}"))
	httpmock.RegisterResponder("GET", fileEndpoint, httpmock.NewStringResponder(404, ""))
	httpmock.RegisterResponder("GET", projectEndpoint, httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("POST", fileEndpoint, httpmock.NewStringResponder(201, "{\n  \"content\": {\n    \"name\": \"test.file\",\n    \"path\": \"test.file\",\n    \"sha\": \"3b18e512dba79e4c8300dd08aeb37f8e728b8dad\",\n    \"type\": \"file\",\n    \"size\": 12,\n    \"encoding\": \"base64\",\n    \"content\": \"aGVsbG8gd29ybGQK\"\n  }\n}"))

	author := config.CommitAuthor{Name: s.owner}
	repo := Repo{Owner: s.owner, Name: s.repoName}
	content := []byte("hello world")
	path := "file.txt"
	message := "add hello world"
	err := s.client.CreateFile(s.ctx, author, repo, content, path, message)
	require.NoError(t, err)
}

func TestGiteaupdateReleaseSuite(t *testing.T) {
	suite.Run(t, new(GiteaupdateReleaseSuite))
}

type GiteaCreateReleaseSuite struct {
	GiteaReleasesTestSuite
}

func (s *GiteaCreateReleaseSuite) TestTemplateError() {
	t := s.T()
	s.ctx.Config.Release.NameTemplate = "{{ .NoKeyLikeThat }}"

	releaseID, err := s.client.CreateRelease(s.ctx, s.description)
	require.Empty(t, releaseID)
	require.Error(t, err)
}

func (s *GiteaCreateReleaseSuite) TestErrorGettingExistingRelease() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(404, ""))

	releaseID, err := s.client.CreateRelease(s.ctx, s.description)
	require.Empty(t, releaseID)
	require.Error(t, err)
}

func (s *GiteaCreateReleaseSuite) TestErrorUpdatingRelease() {
	t := s.T()
	expectedRelease := gitea.Release{TagName: s.tag}
	resp, err := httpmock.NewJsonResponder(200, []gitea.Release{expectedRelease})
	require.NoError(t, err)
	httpmock.RegisterResponder("GET", s.releasesURL, resp)
	httpmock.RegisterResponder("PATCH", s.releaseURL, httpmock.NewStringResponder(400, ""))

	releaseID, err := s.client.CreateRelease(s.ctx, s.description)
	require.Empty(t, releaseID)
	require.Error(t, err)
}

func (s *GiteaCreateReleaseSuite) TestSuccessUpdatingRelease() {
	t := s.T()
	expectedRelease := gitea.Release{
		ID:           666,
		TagName:      s.tag,
		Target:       s.commit,
		Note:         s.description,
		IsDraft:      s.isDraft,
		IsPrerelease: s.isPrerelease,
	}
	resp, err := httpmock.NewJsonResponder(200, []gitea.Release{expectedRelease})
	require.NoError(t, err)
	httpmock.RegisterResponder("GET", s.releasesURL, resp)
	resp, err = httpmock.NewJsonResponder(200, &expectedRelease)
	require.NoError(t, err)
	httpmock.RegisterResponder("PATCH", s.releaseURL, resp)

	newDescription := "NewDescription"
	releaseID, err := s.client.CreateRelease(s.ctx, newDescription)
	require.Equal(t, fmt.Sprint(expectedRelease.ID), releaseID)
	require.NoError(t, err)
}

func (s *GiteaCreateReleaseSuite) TestErrorCreatingRelease() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(200, "[]"))
	httpmock.RegisterResponder("POST", s.releasesURL, httpmock.NewStringResponder(400, ""))

	releaseID, err := s.client.CreateRelease(s.ctx, s.description)
	require.Empty(t, releaseID)
	require.Error(t, err)
}

func (s *GiteaCreateReleaseSuite) TestSuccessCreatingRelease() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(200, "[]"))
	expectedRelease := gitea.Release{
		ID:           666,
		TagName:      s.tag,
		Target:       s.commit,
		Note:         s.description,
		IsDraft:      s.isDraft,
		IsPrerelease: s.isPrerelease,
	}
	resp, err := httpmock.NewJsonResponder(200, &expectedRelease)
	require.NoError(t, err)
	httpmock.RegisterResponder("POST", s.releasesURL, resp)

	releaseID, err := s.client.CreateRelease(s.ctx, s.description)
	require.Equal(t, fmt.Sprint(expectedRelease.ID), releaseID)
	require.NoError(t, err)
}

func TestGiteaCreateReleaseSuite(t *testing.T) {
	suite.Run(t, new(GiteaCreateReleaseSuite))
}

type GiteaUploadSuite struct {
	GiteaReleasesTestSuite
	artifact              *artifact.Artifact
	releaseAttachmentsURL string
}

func (s *GiteaUploadSuite) SetupTest() {
	t := s.T()
	s.GiteaReleasesTestSuite.SetupTest()
	file, err := os.CreateTemp(t.TempDir(), "gitea_test_tempfile")
	require.NoError(t, err)
	require.NotNil(t, file)
	_ = file.Close()
	s.artifact = &artifact.Artifact{Name: "ArtifactName", Path: file.Name()}
	s.releaseAttachmentsURL = fmt.Sprintf("%v/assets", s.releaseURL)
}

func (s *GiteaUploadSuite) TearDownTest() {
	s.GiteaReleasesTestSuite.TearDownTest()
}

func (s *GiteaUploadSuite) TestErrorParsingReleaseID() {
	t := s.T()
	err := s.client.Upload(s.ctx, "notint", s.artifact)
	require.EqualError(t, err, "strconv.ParseInt: parsing \"notint\": invalid syntax")
}

func (s *GiteaUploadSuite) TestErrorCreatingReleaseAttachment() {
	t := s.T()
	httpmock.RegisterResponder("POST", s.releaseAttachmentsURL, httpmock.NewStringResponder(400, ""))

	err := s.client.Upload(s.ctx, fmt.Sprint(s.releaseID), s.artifact)
	require.ErrorContains(t, err, "unknown API error: 400")
}

func (s *GiteaUploadSuite) TestSuccess() {
	t := s.T()
	attachment := gitea.Attachment{}
	resp, err := httpmock.NewJsonResponder(200, &attachment)
	require.NoError(t, err)
	httpmock.RegisterResponder("POST", s.releaseAttachmentsURL, resp)

	err = s.client.Upload(s.ctx, fmt.Sprint(s.releaseID), s.artifact)
	require.NoError(t, err)
}

func TestGiteaUploadSuite(t *testing.T) {
	suite.Run(t, new(GiteaUploadSuite))
}

func TestGiteaReleaseURLTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "string_url",
			downloadURL:     "https://gitea.com",
			wantDownloadURL: "https://gitea.com/owner/name/releases/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITEA_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://gitea.mycompany.com/owner/name/releases/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}",
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

	srv := fakeGitea(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Env: []string{
					"GORELEASER_TEST_GITEA_URLS_DOWNLOAD=https://gitea.mycompany.com",
				},
				GiteaURLs: config.GiteaURLs{
					API:      srv.URL,
					Download: tt.downloadURL,
				},
				Release: config.Release{
					Gitea: config.Repo{
						Owner: "owner",
						Name:  "name",
					},
				},
			})

			client, err := newGitea(ctx, ctx.Token)
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

func TestGiteaGetDefaultBranch(t *testing.T) {
	t.Parallel()
	totalRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		totalRequests++
		defer r.Body.Close()

		if strings.HasSuffix(r.URL.Path, "api/v1/version") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"version\":\"1.12.0\"}")
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})

	client, err := newGitea(ctx, "test-token")
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

func TestGiteaGetDefaultBranchErr(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasSuffix(r.URL.Path, "api/v1/version") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"version\":\"1.12.0\"}")
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "{}")
		}
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})

	client, err := newGitea(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.getDefaultBranch(ctx, repo)
	require.Error(t, err)
}

func TestGiteaChangelog(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasSuffix(r.URL.Path, "api/v1/version") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"version\":\"1.22.0\"}")
		}
		if r.URL.Path == "/api/v1/repos/someone/something/compare/v1.0.0...v1.1.0" {
			bts, err := json.Marshal(gitea.Compare{
				TotalCommits: 1,
				Commits: []*gitea.Commit{
					{
						CommitMeta: &gitea.CommitMeta{
							SHA: "2efbc15d0904f5f966355967a4eedd61a8006660",
						},
						RepoCommit: &gitea.RepoCommit{
							Message: "chore: commit without author",
						},
					},
					{
						CommitMeta: &gitea.CommitMeta{
							SHA: "c8488dc825debca26ade35aefca234b142a515c9",
						},
						Author: &gitea.User{
							UserName: "johndoe",
							FullName: "John Doe",
							Email:    "nope@nope.nope",
						},
						RepoCommit: &gitea.RepoCommit{
							Message: "feat: impl something\n\nnsome other lines",
						},
					},
				},
			})
			assert.NoError(t, err)
			_, err = w.Write(bts)
			assert.NoError(t, err)
		}
	}))
	defer srv.Close()

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})

	client, err := newGitea(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	result, err := client.Changelog(ctx, repo, "v1.0.0", "v1.1.0")
	require.NoError(t, err)
	require.Equal(t, []ChangelogItem{
		{
			SHA:     "2efbc15d0904f5f966355967a4eedd61a8006660",
			Message: "chore: commit without author",
		},
		{
			SHA:     "c8488dc825debca26ade35aefca234b142a515c9",
			Message: "feat: impl something",
			Authors: []Author{{
				Username: "johndoe",
				Name:     "John Doe",
				Email:    "nope@nope.nope",
			}},
			AuthorUsername: "johndoe",
			AuthorName:     "John Doe",
			AuthorEmail:    "nope@nope.nope",
		},
	}, result)
}

func TestGiteatGetInstanceURL(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "http://our.internal.gitea.media/api/v1",
		},
	})

	url, err := getInstanceURL(ctx)
	require.NoError(t, err)
	require.Equal(t, "http://our.internal.gitea.media", url)
}

func TestGiteaGetInstanceURLTemplateError(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: "{{ .NoKeyLikeThat }}"}})
	_, err := getInstanceURL(ctx)
	require.Error(t, err)
}

func TestGiteaGetInstanceURLEmpty(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: ""}})
	_, err := getInstanceURL(ctx)
	require.Error(t, err)
}

func TestGiteaNewGiteaInstanceURLError(t *testing.T) {
	t.Parallel()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: "{{ .NoKeyLikeThat }}"}})
	_, err := newGitea(ctx, "giteatoken")
	require.Error(t, err)
}

func TestGiteaCreateFileNewFile(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"content":{"name":"test.rb","path":"test.rb"}}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.NoError(t, err)
}

func TestGiteaCreateFileUpdateExisting(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"name":"test.rb","path":"test.rb","sha":"abc123","type":"file","content":"Y29udGVudA=="}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodPut:
			fmt.Fprint(w, `{"content":{"name":"test.rb","path":"test.rb"}}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo", Branch: "main"}, []byte("new content"), "test.rb", "update test")
	require.NoError(t, err)
}

func TestGiteaCreateFileGetContentsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.Error(t, err)
}

func TestGiteaCreateFileUpdateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"name":"test.rb","path":"test.rb","sha":"abc123","type":"file","content":"Y29udGVudA=="}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo", Branch: "main"}, []byte("updated"), "test.rb", "update test")
	require.Error(t, err)
}

func TestGiteaCreateFileCreateError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo", Branch: "main"}, []byte("content"), "test.rb", "add test")
	require.Error(t, err)
}

func TestGiteaCreateFileDefaultBranchFallback(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch {
		case r.URL.Path == "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.12.0"}`)
		case r.URL.Path == "/api/v1/repos/owner/repo" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"not found"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/repos/owner/repo/contents/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"content":{"name":"test.rb","path":"test.rb"}}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{}")
		}
	}))
	t.Cleanup(srv.Close)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{GiteaURLs: config.GiteaURLs{API: srv.URL}})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "user", Email: "u@e.com"}, Repo{Owner: "owner", Name: "repo"}, []byte("content"), "test.rb", "add test")
	require.NoError(t, err)
}

func TestGiteaCreateFileDefaultBranchError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v1/version" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"version":"1.12.0"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/repos/someone/something") && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"server error"}`)
			return
		}
		if strings.Contains(r.URL.Path, "/contents/newfile.txt") {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"content":{"name":"newfile.txt","path":"newfile.txt","sha":"abc123"}}`)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{API: srv.URL},
	})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)

	// No branch specified, getDefaultBranch will fail, falls back to server default
	repo := Repo{Owner: "someone", Name: "something"}
	err = client.CreateFile(ctx, config.CommitAuthor{Name: "test", Email: "test@test.com"}, repo, []byte("content"), "newfile.txt", "test commit")
	require.NoError(t, err)
}

func TestGiteaCloseMilestoneSuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v1/version" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"version":"1.12.0"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/milestones") && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":1,"title":"v1.0.0","state":"open"}]`)
			return
		}
		if strings.Contains(r.URL.Path, "/milestones/") && r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id":1,"title":"v1.0.0","state":"closed"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{API: srv.URL},
	})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)

	repo := Repo{Owner: "someone", Name: "something"}
	err = client.CloseMilestone(ctx, repo, "v1.0.0")
	require.NoError(t, err)
}

func TestGiteaCloseMilestoneNotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.URL.Path == "/api/v1/version" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"version":"1.12.0"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/milestones") && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"id":1,"title":"v1.0.0","state":"open"}]`)
			return
		}
		if strings.Contains(r.URL.Path, "/milestones/") && r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"milestone not found"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	}))
	t.Cleanup(srv.Close)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		GiteaURLs: config.GiteaURLs{API: srv.URL},
	})
	client, err := newGitea(ctx, "giteatoken")
	require.NoError(t, err)

	repo := Repo{Owner: "someone", Name: "something"}
	err = client.CloseMilestone(ctx, repo, "v1.0.0")
	require.ErrorAs(t, err, &ErrNoMilestoneFound{})
}

func TestGiteaPublishRelease(t *testing.T) {
	t.Parallel()
	client := &giteaClient{}
	ctx := testctx.Wrap(t.Context())
	require.NoError(t, client.PublishRelease(ctx, "123"))
}
