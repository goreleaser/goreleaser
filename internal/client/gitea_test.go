package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"text/template"

	"code.gitea.io/sdk/gitea"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GetInstanceURLSuite struct {
	suite.Suite
}

func (s *GetInstanceURLSuite) TestWithScheme() {
	t := s.T()
	rootURL := "https://gitea.com"
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "{{ .Env.GORELEASER_NOT_EXISTS }}",
		},
	})

	result, err := getInstanceURL(ctx)
	require.ErrorAs(t, err, &template.ExecError{})
	require.Empty(t, result)
}

func (s *GetInstanceURLSuite) TestTemplateInvalid() {
	t := s.T()
	ctx := context.New(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: "{{.dddddddddd",
		},
	})

	result, err := getInstanceURL(ctx)
	require.Error(t, err)
	require.Empty(t, result)
}

func TestGetInstanceURLSuite(t *testing.T) {
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
	s.ctx = &context.Context{
		Version: "6.6.6",
		Config: config.Project{
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
		Env: context.Env{},
		Semver: context.Semver{
			Major: 6,
			Minor: 6,
			Patch: 6,
		},
		Git: context.GitInfo{
			CurrentTag:  s.tag,
			Commit:      s.commit,
			ShortCommit: s.commit[0:2],
			URL:         "https://gitea.com/goreleaser/goreleaser.git",
		},
		PreRelease: s.isPrerelease,
	}
	s.releaseID = 666
	s.releaseURL = fmt.Sprintf("%v/%v", s.releasesURL, s.releaseID)
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s/api/v1/version", s.url), httpmock.NewStringResponder(200, "{\"version\":\"1.12.0\"}"))
	newClient, err := gitea.NewClient(s.url)
	require.NoError(s.T(), err)
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

	release, err := s.client.getExistingRelease(s.owner, s.repoName, s.tag)
	require.Nil(t, release)
	require.NoError(t, err)
}

func (s *GetExistingReleaseSuite) TestNoRepo() {
	t := s.T()
	httpmock.RegisterResponder("GET", s.releasesURL, httpmock.NewStringResponder(404, ""))

	release, err := s.client.getExistingRelease(s.owner, s.repoName, s.tag)
	require.Nil(t, release)
	require.Error(t, err)
}

func (s *GetExistingReleaseSuite) TestReleaseExists() {
	t := s.T()
	release := gitea.Release{TagName: s.tag}
	resp, err := httpmock.NewJsonResponder(200, []gitea.Release{release})
	require.NoError(t, err)
	httpmock.RegisterResponder("GET", s.releasesURL, resp)

	result, err := s.client.getExistingRelease(s.owner, s.repoName, s.tag)
	require.NotNil(t, result)
	require.Equal(t, *result, release)
	require.NoError(t, err)
}

func TestGetExistingReleaseSuite(t *testing.T) {
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

func (s *GiteaupdateReleaseSuite) SetupTest() {
	s.GiteaReleasesTestSuite.SetupTest()
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
	require.Nil(t, err)
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

func (s *GiteaCreateReleaseSuite) TestErrorGettingExisitngRelease() {
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
	file                  *os.File
	releaseAttachmentsURL string
}

func (s *GiteaUploadSuite) SetupTest() {
	t := s.T()
	s.GiteaReleasesTestSuite.SetupTest()
	s.artifact = &artifact.Artifact{Name: "ArtifactName"}
	file, err := ioutil.TempFile(t.TempDir(), "gitea_test_tempfile")
	require.NoError(t, err)
	require.NotNil(t, file)
	t.Cleanup(func() {
		_ = file.Close()
	})
	s.file = file
	s.releaseAttachmentsURL = fmt.Sprintf("%v/assets", s.releaseURL)
}

func (s *GiteaUploadSuite) TearDownTest() {
	s.GiteaReleasesTestSuite.TearDownTest()
	require.NoError(s.T(), s.file.Close())
}

func (s *GiteaUploadSuite) TestErrorParsingReleaseID() {
	t := s.T()
	err := s.client.Upload(s.ctx, "notint", s.artifact, s.file)
	require.EqualError(t, err, "strconv.ParseInt: parsing \"notint\": invalid syntax")
}

func (s *GiteaUploadSuite) TestErrorCreatingReleaseAttachment() {
	t := s.T()
	httpmock.RegisterResponder("POST", s.releaseAttachmentsURL, httpmock.NewStringResponder(400, ""))

	err := s.client.Upload(s.ctx, fmt.Sprint(s.releaseID), s.artifact, s.file)
	require.True(t, strings.HasPrefix(err.Error(), "Unknown API Error: 400"))
}

func (s *GiteaUploadSuite) TestSuccess() {
	t := s.T()
	attachment := gitea.Attachment{}
	resp, err := httpmock.NewJsonResponder(200, &attachment)
	require.NoError(t, err)
	httpmock.RegisterResponder("POST", s.releaseAttachmentsURL, resp)

	err = s.client.Upload(s.ctx, fmt.Sprint(s.releaseID), s.artifact, s.file)
	require.NoError(t, err)
}

func TestGiteaUploadSuite(t *testing.T) {
	suite.Run(t, new(GiteaUploadSuite))
}

func TestGiteaReleaseURLTemplate(t *testing.T) {
	tests := []struct {
		name            string
		downloadURL     string
		wantDownloadURL string
		wantErr         bool
	}{
		{
			name:            "string_url",
			downloadURL:     "https://gitea.com",
			wantDownloadURL: "https://gitea.com/owner/name/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
		},
		{
			name:            "download_url_template",
			downloadURL:     "{{ .Env.GORELEASER_TEST_GITEA_URLS_DOWNLOAD }}",
			wantDownloadURL: "https://gitea.mycompany.com/owner/name/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
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
					"GORELEASER_TEST_GITEA_URLS_DOWNLOAD=https://gitea.mycompany.com",
				},
				GiteaURLs: config.GiteaURLs{
					API:      "https://gitea.com/api/v1",
					Download: tt.downloadURL,
				},
				Release: config.Release{
					Gitea: config.Repo{
						Owner: "owner",
						Name:  "name",
					},
				},
			})
			client, err := NewGitea(ctx, ctx.Token)
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

	ctx := context.New(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})
	client, err := NewGitea(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.GetDefaultBranch(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 2, totalRequests)
}

func TestGiteaGetDefaultBranchErr(t *testing.T) {
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

	ctx := context.New(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})
	client, err := NewGitea(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.GetDefaultBranch(ctx, repo)
	require.Error(t, err)
}

func TestGiteaChangelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if strings.HasSuffix(r.URL.Path, "api/v1/version") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"version\":\"1.12.0\"}")
		}
	}))
	defer srv.Close()

	ctx := context.New(config.Project{
		GiteaURLs: config.GiteaURLs{
			API: srv.URL,
		},
	})
	client, err := NewGitea(ctx, "test-token")
	require.NoError(t, err)
	repo := Repo{
		Owner:  "someone",
		Name:   "something",
		Branch: "somebranch",
	}

	_, err = client.Changelog(ctx, repo, "v1.0.0", "v1.1.0")
	require.EqualError(t, err, ErrNotImplemented.Error())
}
