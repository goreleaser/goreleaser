package client

import (
	"errors"
	"os"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func NewMock() *Mock {
	return &Mock{}
}

type Mock struct {
	CreatedFile bool
	Content     string
	Path        string
}

func (c *Mock) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	return nil
}

func (c *Mock) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	return "", errors.New("Mock does not yet implement GetDefaultBranch")
}

func (c *Mock) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return
}

func (c *Mock) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return "https://dummyhost/download/{{ .Tag }}/{{ .ArtifactName }}", nil
}

func (c *Mock) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path, msg string) (err error) {
	c.CreatedFile = true
	c.Content = string(content)
	c.Path = path
	return
}

func (c *Mock) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error) {
	return
}
