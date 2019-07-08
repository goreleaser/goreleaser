package client

import (
	"fmt"
	"testing"

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
