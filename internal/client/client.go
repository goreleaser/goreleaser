// Package client contains the client implementations for several providers.
package client

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrNotImplemented is returned when a client does not implement certain feature.
var ErrNotImplemented = fmt.Errorf("not implemented")

// Info of the repository.
type Info struct {
	Description string
	Homepage    string
	URL         string
}

type Repo struct {
	Owner  string
	Name   string
	Branch string
}

func (r Repo) String() string {
	if r.Owner == "" && r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// Client interface.
type Client interface {
	CloseMilestone(ctx *context.Context, repo Repo, title string) (err error)
	CreateRelease(ctx *context.Context, body string) (releaseID string, err error)
	ReleaseURLTemplate(ctx *context.Context) (string, error)
	CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path, message string) (err error)
	Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error)
	GetDefaultBranch(ctx *context.Context, repo Repo) (string, error)
	Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error)
}

// New creates a new client depending on the token type.
func New(ctx *context.Context) (Client, error) {
	return newWithToken(ctx, ctx.Token)
}

func newWithToken(ctx *context.Context, token string) (Client, error) {
	log.WithField("type", ctx.TokenType).Debug("token type")
	switch ctx.TokenType {
	case context.TokenTypeGitHub:
		return NewGitHub(ctx, token)
	case context.TokenTypeGitLab:
		return NewGitLab(ctx, token)
	case context.TokenTypeGitea:
		return NewGitea(ctx, token)
	default:
		return nil, fmt.Errorf("invalid client token type: %q", ctx.TokenType)
	}
}

func NewIfToken(ctx *context.Context, cli Client, token string) (Client, error) {
	if token == "" {
		return cli, nil
	}
	token, err := tmpl.New(ctx).ApplySingleEnvOnly(token)
	if err != nil {
		return nil, err
	}
	log.Debug("using custom token")
	return newWithToken(ctx, token)
}

// ErrNoMilestoneFound is an error when no milestone is found.
type ErrNoMilestoneFound struct {
	Title string
}

func (e ErrNoMilestoneFound) Error() string {
	return fmt.Sprintf("no milestone found: %s", e.Title)
}

// RetriableError is an error that will cause the action to be retried.
type RetriableError struct {
	Err error
}

func (e RetriableError) Error() string {
	return e.Err.Error()
}
