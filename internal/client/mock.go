package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var (
	_ Client                = &Mock{}
	_ ReleaseNotesGenerator = &Mock{}
	_ PullRequestOpener     = &Mock{}
	_ ForkSyncer            = &Mock{}
)

// NewMock creates a new mock client.
func NewMock() *Mock {
	return &Mock{}
}

// Mock is a mock client.
type Mock struct {
	CreatedFile          bool
	Content              string
	Path                 string
	Messages             []string
	FailToCreateRelease  bool
	FailToUpload         bool
	CreatedRelease       bool
	UploadedFile         bool
	ReleasePublished     bool
	UploadedFileNames    []string
	UploadedFilePaths    map[string]string
	FailFirstUpload      bool
	Lock                 sync.Mutex
	ClosedMilestone      string
	FailToCloseMilestone bool
	Changes              []ChangelogItem
	ReleaseNotes         string
	ReleaseNotesParams   []string
	OpenedPullRequest    bool
	SyncedFork           bool
}

// SyncFork implements Client.
func (c *Mock) SyncFork(_ *context.Context, _ Repo, _ Repo) error {
	c.SyncedFork = true
	return nil
}

// OpenPullRequest implements PullRequestOpener.
func (c *Mock) OpenPullRequest(_ *context.Context, _, _ Repo, _ string, _ bool) error {
	c.OpenedPullRequest = true
	return nil
}

// Changelog implements Client.
func (c *Mock) Changelog(_ *context.Context, _ Repo, _, _ string) ([]ChangelogItem, error) {
	if len(c.Changes) > 0 {
		return c.Changes, nil
	}
	return nil, ErrNotImplemented
}

// GenerateReleaseNotes implements ReleaseNotesGenerator.
func (c *Mock) GenerateReleaseNotes(_ *context.Context, _ Repo, prev, current string) (string, error) {
	if c.ReleaseNotes != "" {
		c.ReleaseNotesParams = []string{prev, current}
		return c.ReleaseNotes, nil
	}
	return "", ErrNotImplemented
}

// CloseMilestone implements Client.
func (c *Mock) CloseMilestone(_ *context.Context, _ Repo, title string) error {
	if c.FailToCloseMilestone {
		return errors.New("milestone failed")
	}

	c.ClosedMilestone = title

	return nil
}

// CreateRelease implements Client.
func (c *Mock) CreateRelease(_ *context.Context, _ string) (string, error) {
	if c.FailToCreateRelease {
		return "", errors.New("release failed")
	}
	c.CreatedRelease = true
	return "", nil
}

// PublishRelease implements Client.
func (c *Mock) PublishRelease(_ *context.Context, _ string /* releaseID */) (err error) {
	c.ReleasePublished = true
	return nil
}

// ReleaseURLTemplate implements Client.
func (c *Mock) ReleaseURLTemplate(_ *context.Context) (string, error) {
	return "https://dummyhost/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}", nil
}

// CreateFile implements Client.
func (c *Mock) CreateFile(_ *context.Context, _ config.CommitAuthor, _ Repo, content []byte, path, msg string) error {
	c.CreatedFile = true
	c.Content = string(content)
	c.Path = path
	c.Messages = append(c.Messages, msg)
	return nil
}

// Upload implements Client.
func (c *Mock) Upload(_ *context.Context, _ string, artifact *artifact.Artifact, file *os.File) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if c.UploadedFilePaths == nil {
		c.UploadedFilePaths = map[string]string{}
	}
	// ensure file is read to better mimic real behavior
	_, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}
	if c.FailToUpload {
		return errors.New("upload failed")
	}
	if c.FailFirstUpload {
		c.FailFirstUpload = false
		return RetriableError{Err: errors.New("upload failed, should retry")}
	}
	c.UploadedFile = true
	c.UploadedFileNames = append(c.UploadedFileNames, artifact.Name)
	c.UploadedFilePaths[artifact.Name] = artifact.Path
	return nil
}
