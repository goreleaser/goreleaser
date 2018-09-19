package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	assert.Equal(
		t,
		"goreleaser/godownloader",
		Repo{Owner: "goreleaser", Name: "godownloader"}.String(),
	)
}

func TestEmptyRepoNameAndOwner(t *testing.T) {
	assert.Empty(t, Repo{}.String())
}

func TestLoadReader(t *testing.T) {
	var conf = `
nfpm:
  homepage: http://goreleaser.github.io
`
	buf := strings.NewReader(conf)
	prop, err := LoadReader(buf)

	assert.NoError(t, err)
	assert.Equal(t, "http://goreleaser.github.io", prop.NFPM.Homepage, "yaml did not load correctly")
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 1, fmt.Errorf("error")
}
func TestLoadBadReader(t *testing.T) {
	_, err := LoadReader(errorReader{})
	assert.Error(t, err)
}

func TestFile(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "config")
	assert.NoError(t, err)
	_, err = Load(filepath.Join(f.Name()))
	assert.NoError(t, err)
}

func TestFileNotFound(t *testing.T) {
	_, err := Load("/nope/no-way.yml")
	assert.Error(t, err)
}

func TestInvalidFields(t *testing.T) {
	_, err := Load("testdata/invalid_config.yml")
	assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 2: field invalid_yaml not found in type config.Build")
}

func TestInvalidYaml(t *testing.T) {
	_, err := Load("testdata/invalid.yml")
	assert.EqualError(t, err, "yaml: line 1: did not find expected node content")
}

func TestConfigWithAnchors(t *testing.T) {
	_, err := Load("testdata/anchor.yaml")
	assert.NoError(t, err)
}

func TestHomebrewWithGithub(t *testing.T) {
	prop, err := Load("testdata/homebrew_github.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "homebrew with github", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "brew project", prop.Brew.Name, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Brew.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Brew.Repo.Name, "yaml repo did not load correctly")
}

func TestHomebrewWithGitlab(t *testing.T) {
	prop, err := Load("testdata/homebrew_gitlab.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "homebrew with gitlab", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "brew project", prop.Brew.Name, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Brew.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Brew.Repo.Name, "yaml repo did not load correctly")
}

func TestHomebrewWithGithubAndGitlab(t *testing.T) {
	_, err := Load("testdata/homebrew_github_and_gitlab.yaml")
	assert.EqualError(t, err, "homebrew: cannot define both github and gitlab")
}

func TestScoopWithBucket(t *testing.T) {
	prop, err := Load("testdata/scoop_bucket.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "scoop with bucket", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "scoop homepage", prop.Scoop.Homepage, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Scoop.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Scoop.Repo.Name, "yaml repo did not load correctly")
}

func TestScoopWithGithub(t *testing.T) {
	prop, err := Load("testdata/scoop_github.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "scoop with github", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "scoop homepage", prop.Scoop.Homepage, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Scoop.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Scoop.Repo.Name, "yaml repo did not load correctly")
}

func TestScoopWithGitlab(t *testing.T) {
	prop, err := Load("testdata/scoop_gitlab.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "scoop with gitlab", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "scoop homepage", prop.Scoop.Homepage, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Scoop.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Scoop.Repo.Name, "yaml repo did not load correctly")
}

func TestScoopWithGithubAndBucket(t *testing.T) {
	_, err := Load("testdata/scoop_github_and_bucket.yaml")
	assert.EqualError(t, err, "scoop: cannot define multiple github, gitlab, and bucket")
}

func TestScoopWithGithubAndGitlab(t *testing.T) {
	_, err := Load("testdata/scoop_github_and_gitlab.yaml")
	assert.EqualError(t, err, "scoop: cannot define multiple github, gitlab, and bucket")
}

func TestScoopWithGitlabAndBucket(t *testing.T) {
	_, err := Load("testdata/scoop_gitlab_and_bucket.yaml")
	assert.EqualError(t, err, "scoop: cannot define multiple github, gitlab, and bucket")
}

func TestReleaseWithGithub(t *testing.T) {
	prop, err := Load("testdata/release_github.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "release with github", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Release.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Release.Repo.Name, "yaml repo did not load correctly")
}

func TestReleaseWithGitlab(t *testing.T) {
	prop, err := Load("testdata/release_gitlab.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "release with gitlab", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "foo", prop.Release.Repo.Owner, "yaml repo did not load correctly")
	assert.Equal(t, "bar", prop.Release.Repo.Name, "yaml repo did not load correctly")
}

func TestReleaseWithGithubAndGitlab(t *testing.T) {
	_, err := Load("testdata/release_github_and_gitlab.yaml")
	assert.EqualError(t, err, "release: cannot define both github and gitlab")
}

func TestProjectWithGithub(t *testing.T) {
	prop, err := Load("testdata/github_urls.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "project with github urls", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "https://git.company.com/api/v3/", prop.RepoURLs.API, "yaml urls did not load correctly")
	assert.Equal(t, "https://git.company.com/api/uploads/", prop.RepoURLs.Upload, "yaml urls did not load correctly")
	assert.Equal(t, "https://git.company.com/", prop.RepoURLs.Download, "yaml urls did not load correctly")
}

func TestProjectWithGitlab(t *testing.T) {
	prop, err := Load("testdata/gitlab_urls.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "project with gitlab urls", prop.ProjectName, "yaml did not load correctly")
	assert.Equal(t, "https://git.company.com/api/v3/", prop.RepoURLs.API, "yaml urls did not load correctly")
	assert.Equal(t, "https://git.company.com/api/uploads/", prop.RepoURLs.Upload, "yaml urls did not load correctly")
	assert.Equal(t, "https://git.company.com/", prop.RepoURLs.Download, "yaml urls did not load correctly")
}

func TestProjectWithGithubAndGitlab(t *testing.T) {
	_, err := Load("testdata/github_and_gitlab_urls.yaml")
	assert.EqualError(t, err, "project: cannot define both github and gitlab urls")
}
