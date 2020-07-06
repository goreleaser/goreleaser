package client

import (
	"fmt"
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
)

func TestExtractHashFromProjectFileURL(t *testing.T) {
	givenHash := "22e8b1508b0f28433b94754a5ea2f4aa"
	projectFileURL := fmt.Sprintf("/uploads/%s/release-testing_0.3.7_Darwin_x86_64.tar.gz", givenHash)
	extractedHash, err := extractProjectFileHashFrom(projectFileURL)
	if err != nil {
		t.Errorf("expexted no error but got: %v", err)
	}
	assert.Equal(t, givenHash, extractedHash)
}

func TestFailToExtractHashFromProjectFileURL(t *testing.T) {
	givenHash := "22e8b1508b0f28433b94754a5ea2f4aa"
	projectFileURL := fmt.Sprintf("/uploads/%s/new-path/file.ext", givenHash)
	_, err := extractProjectFileHashFrom(projectFileURL)
	if err == nil {
		t.Errorf("expected an error but got none for new-path in url")
	}

	projectFileURL = fmt.Sprintf("/%s/file.ext", givenHash)
	_, err = extractProjectFileHashFrom(projectFileURL)
	if err == nil {
		t.Errorf("expected an error but got none for path-too-small in url")
	}
}

func TestGitLabReleaseURLTemplate(t *testing.T) {
	var ctx = context.New(config.Project{
		GitLabURLs: config.GitLabURLs{
			// default URL would otherwise be set via pipe/defaults
			Download: DefaultGitLabDownloadURL,
		},
		Release: config.Release{
			GitLab: config.Repo{
				Owner: "owner",
				Name:  "name",
			},
		},
	})
	client, err := NewGitLab(ctx)
	assert.NoError(t, err)

	urlTpl, err := client.ReleaseURLTemplate(ctx)
	assert.NoError(t, err)

	expectedUrl := "https://gitlab.com/owner/name/uploads/{{ .ArtifactUploadHash }}/{{ .ArtifactName }}"
	assert.Equal(t, expectedUrl, urlTpl)
}
