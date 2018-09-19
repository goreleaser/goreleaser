package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type gitlabClient struct {
	AccessToken string
	BaseURL     string
	HTTPClient  *http.Client
	uploadLock  sync.Mutex
}

// NewGitLab returns a gitlab client implementation
func NewGitLab(ctx *context.Context) (*gitlabClient, error) {
	if ctx.Config.RepoURLs.API == "" {
		ctx.Config.RepoURLs.API = "https://gitlab.com/api/v4"
	}
	if ctx.Config.RepoURLs.Download == "" {
		ctx.Config.RepoURLs.Download = "https://gitlab.com/"
	}

	client := &gitlabClient{
		AccessToken: ctx.StorageToken,
		HTTPClient:  &http.Client{},
	}
	api, err := url.Parse(ctx.Config.RepoURLs.API)
	if err != nil {
		return &gitlabClient{}, err
	}
	client.BaseURL = api.String()

	return client, nil
}

func (c *gitlabClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo config.Repo,
	content bytes.Buffer,
	p string,
	message string,
) error {

	u := fmt.Sprintf(
		"%s/projects/%s/repository/files/%s?ref=master",
		c.BaseURL,
		projectID(repo.Owner, repo.Name),
		url.QueryEscape(p),
	)

	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return fmt.Errorf("gitlab get file: error creating new HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

	resp, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("gitlab get file: error executing HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if err != nil && resp.StatusCode != http.StatusNotFound {
		return err
	}

	opts := struct {
		Branch        string `json:"branch"`
		Content       string `json:"content"`
		CommitMessage string `json:"commit_message"`
	}{
		Branch:        "master",
		Content:       content.String(),
		CommitMessage: message,
	}
	d, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("gitlab create/update file: error JSON marshaling options: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {

		req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(d))
		if err != nil {
			return fmt.Errorf("gitlab create file: error creating new HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

		resp, err := c.HTTPClient.Do(req.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("gitlab create file: error executing HTTP request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("gitlab create file: unexpected HTTP status code: got %d; want %d", resp.StatusCode, http.StatusOK)
		}
	} else {

		req, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(d))
		if err != nil {
			return fmt.Errorf("gitlab update file: error creating new HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

		resp, err := c.HTTPClient.Do(req.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("gitlab update file: error executing HTTP request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("gitlab update file: unexpected HTTP status code: got %d; want %d", resp.StatusCode, http.StatusOK)
		}
	}
	return nil
}

func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (string, error) {

	var method string
	if _, err := c.getReleaseNotes(ctx, ctx.Git.CurrentTag); err == errReleaseNotFound {
		method = http.MethodPost
	} else {
		method = http.MethodPut
	}

	return ctx.Git.CurrentTag, c.sendReleaseNotes(ctx, method, ctx.Git.CurrentTag, body)
}

func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID string,
	name string,
	file *os.File,
) (string, error) {
	markdown, u, err := c.uploadFile(ctx, name, file)
	if err != nil {
		return "", err
	}

	// We lock the mutex so can append to the release description using a get and put
	c.uploadLock.Lock()
	defer c.uploadLock.Unlock()
	notes, err := c.getReleaseNotes(ctx, releaseID)
	if err != nil {
		return "", err
	}
	notes = fmt.Sprintf("%s\n\n%s", notes, markdown)
	if err := c.sendReleaseNotes(ctx, http.MethodPut, releaseID, notes); err != nil {
		return "", err
	}

	return u, nil
}

func (c *gitlabClient) sendReleaseNotes(
	ctx *context.Context,
	method string,
	releaseID string,
	body string,
) error {
	u := fmt.Sprintf(
		"%s/projects/%s/repository/tags/%s/release",
		c.BaseURL,
		projectID(ctx.Config.Release.Repo.Owner, ctx.Config.Release.Repo.Name),
		releaseID,
	)
	opts := struct {
		Description string `json:"description"`
	}{
		Description: body,
	}
	d, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("gitlab %s release: error JSON marshaling options: %v", method, err)
	}

	req, err := http.NewRequest(method, u, bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("gitlab %s release: error creating new HTTP request: %v", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

	resp, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("gitlab %s release: error executing HTTP request: %v", method, err)
	}
	defer resp.Body.Close()

	var statusCode int
	if method == http.MethodPost {
		statusCode = http.StatusCreated
	} else {
		statusCode = http.StatusOK
	}

	if resp.StatusCode != statusCode {
		return fmt.Errorf("gitlab %s release: unexpected HTTP status code: got %d; want %d", method, resp.StatusCode, statusCode)
	}
	return nil
}

var errReleaseNotFound = errors.New("release not found")

func (c *gitlabClient) getReleaseNotes(
	ctx *context.Context,
	releaseID string,
) (string, error) {
	u := fmt.Sprintf(
		"%s/projects/%s/repository/tags/%s",
		c.BaseURL,
		projectID(ctx.Config.Release.Repo.Owner,
			ctx.Config.Release.Repo.Name),
		releaseID,
	)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab release notes: error creating new HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

	resp, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("gitlab release notes: error executing HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gitlab release notes: unexpected HTTP status code: got %d; want %d", resp.StatusCode, http.StatusOK)
	}

	var respBody struct {
		Release *struct {
			Description string `json:"description"`
		} `json:"release"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("gitlab release notes: error JSON decoding HTTP response: %v", err)
	}

	if respBody.Release == nil {
		return "", errReleaseNotFound
	}

	return respBody.Release.Description, nil
}

func (c *gitlabClient) uploadFile(
	ctx *context.Context,
	name string,
	file *os.File,
) (string, string, error) {
	u := fmt.Sprintf(
		"%s/projects/%s/uploads",
		c.BaseURL,
		projectID(ctx.Config.Release.Repo.Owner, ctx.Config.Release.Repo.Name),
	)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return "", "", err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return "", "", fmt.Errorf("gitlab upload file: error creating new HTTP request: %v", err)
	}
	req.ContentLength = int64(body.Len())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("PRIVATE-TOKEN", c.AccessToken)

	resp, err := c.HTTPClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", "", fmt.Errorf("gitlab upload file: error executing HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("gitlab upload file: unexpected HTTP status code: got %d; want %d", resp.StatusCode, http.StatusOK)
	}

	var respBody struct {
		URL      string `json:"url"`
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", "", fmt.Errorf("gitlab upload file: error JSON decoding HTTP response: %v", err)
	}
	downloadURL := path.Clean(
		fmt.Sprintf(
			"/%s/%s/%s",
			ctx.Config.Release.Repo.Owner,
			ctx.Config.Release.Repo.Name,
			respBody.URL,
		),
	)
	return respBody.Markdown, downloadURL, nil
}

func projectID(owner, name string) string {
	return url.QueryEscape(fmt.Sprintf("%s/%s", owner, name))
}
