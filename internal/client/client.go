// Package client contains the client implementations for several providers.
package client

import (
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
	CreateRelease(ctx *context.Context, body string) (releaseID int64, err error)
	CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo config.Repo, content []byte, path, message string) (err error)
	Upload(ctx *context.Context, releaseID int64, name string, file *os.File) (err error)
}
