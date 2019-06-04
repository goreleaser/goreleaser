// Package client contains the client implementations for several providers.
package client

import (
	"bytes"
	"os"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Info of the repository
type Info struct {
	Description string
	Homepage    string
	URL         string
}

// Client interface
type Client interface {
	CreateRelease(ctx *context.Context, body string) (releaseID string, err error)
	CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content bytes.Buffer, path, message string) (err error)
	Upload(ctx *context.Context, releaseID string, name string, file *os.File) (err error)
}
