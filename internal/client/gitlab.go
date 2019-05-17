package client

import (
	"bytes"
	"os"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/xanzy/go-gitlab"
)

type gitlabClient struct {
	client *gitlab.Client
}

// NewGitLab returns a gitlab client implementation
func NewGitLab(ctx *context.Context) (Client, error) {
	token := "xx" // ctx.Token
	client := gitlab.NewClient(nil, token)
	if ctx.Config.GitLabURLs.API != "" {
		err := client.SetBaseURL(ctx.Config.GitLabURLs.API)
		if err != nil {
			return &gitlabClient{}, err
		}
	}
	return &gitlabClient{client: client}, nil
}

func (c *gitlabClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo config.Repo,
	content bytes.Buffer,
	path,
	message string,
) error {
	projectID := ""
	fileName := ""
	// options := nil
	_, _, err := c.client.Projects.UploadFile(projectID, fileName, nil)

	return err
}

func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (releaseID int64, err error) {
	return int64(1), nil
}

func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID int64,
	name string,
	file *os.File,
) error {
	// c.client.Releases.
	return nil
}
