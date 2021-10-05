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
}

func (c *Mock) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	return "", ErrNotImplemented
}

func (c *Mock) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	if c.FailToCloseMilestone {
		return errors.New("milestone failed")
	}

	c.ClosedMilestone = title

	return nil
}

func (c *Mock) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	return "", ErrNotImplemented
}

func (c *Mock) CreateRelease(ctx *context.Context, body string) (string, error) {
	if c.FailToCreateRelease {
		return "", errors.New("release failed")
	}
	c.CreatedRelease = true
	return "", nil
}

func (c *Mock) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "https://dummyhost/download/{{ .Tag }}/{{ .ArtifactName }}", nil
}

func (c *Mock) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path, msg string) error {
	c.CreatedFile = true
	c.Content = string(content)
	c.Path = path
	return nil
}

func (c *Mock) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) error {
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
