package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

var (
	_ Client                = &Mock{}
	_ ReleaseNotesGenerator = &Mock{}
	_ PullRequestOpener     = &Mock{}
)

func NewMock() *Mock {
	return &Mock{}
}

type Mock struct {
	CreatedFile          bool
	Content              string
	Path                 string
	FailToCreateRelease  bool
	FailToUpload         bool
	CreatedRelease       bool
	UploadedFile         bool
	UploadedFileNames    []string
	UploadedFilePaths    map[string]string
	FailFirstUpload      bool
	Lock                 sync.Mutex
	ClosedMilestone      string
	FailToCloseMilestone bool
	Changes              string
	ReleaseNotes         string
	ReleaseNotesParams   []string
	OpenedPullRequest    bool
}

func (c *Mock) OpenPullRequest(_ *context.Context, _ Repo, _, _ string) error {
	c.OpenedPullRequest = true
	return nil
}

func (c *Mock) Changelog(_ *context.Context, _ Repo, _, _ string) (string, error) {
	if c.Changes != "" {
		return c.Changes, nil
	}
	return "", ErrNotImplemented
}

func (c *Mock) GenerateReleaseNotes(_ *context.Context, _ Repo, prev, current string) (string, error) {
	if c.ReleaseNotes != "" {
		c.ReleaseNotesParams = []string{prev, current}
		return c.ReleaseNotes, nil
	}
	return "", ErrNotImplemented
}

func (c *Mock) CloseMilestone(_ *context.Context, _ Repo, title string) error {
	if c.FailToCloseMilestone {
		return errors.New("milestone failed")
	}

	c.ClosedMilestone = title

	return nil
}

func (c *Mock) CreateRelease(_ *context.Context, _ string) (string, error) {
	if c.FailToCreateRelease {
		return "", errors.New("release failed")
	}
	c.CreatedRelease = true
	return "", nil
}

func (c *Mock) ReleaseURLTemplate(_ *context.Context) (string, error) {
	return "https://dummyhost/download/{{ .Tag }}/{{ .ArtifactName }}", nil
}

func (c *Mock) CreateFile(_ *context.Context, _ config.CommitAuthor, _ Repo, content []byte, path, _ string) error {
	c.CreatedFile = true
	c.Content = string(content)
	c.Path = path
	return nil
}

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
