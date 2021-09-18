// Package client contains the client implementations for several providers.
package client

import (
	"errors"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Info of the repository.
type Info struct {
	Description string
	Homepage    string
	URL         string
}

type Repo struct {
	Owner string
	Name  string
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
}

// New creates a new client depending on the token type.
func New(ctx *context.Context) (Client, error) {
	log.WithField("type", ctx.TokenType).Debug("token type")
	if ctx.TokenType == context.TokenTypeGitHub {
		return NewGitHub(ctx, ctx.Token)
	}
	if ctx.TokenType == context.TokenTypeGitLab {
		return NewGitLab(ctx, ctx.Token)
	}
	if ctx.TokenType == context.TokenTypeGitea {
		return NewGitea(ctx, ctx.Token)
	}
	return nil, nil
}

func newWithToken(ctx *context.Context, token string) (Client, error) {
	if ctx.TokenType == context.TokenTypeGitHub {
		return NewGitHub(ctx, token)
	}
	if ctx.TokenType == context.TokenTypeGitLab {
		return NewGitLab(ctx, token)
	}
	if ctx.TokenType == context.TokenTypeGitea {
		return NewGitea(ctx, token)
	}
	return nil, nil
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

// NotImplementedError happens when trying to use something a client does not
// implement.
type NotImplementedError struct {
	TokenType context.TokenType
}

func (e NotImplementedError) Error() string {
	return fmt.Sprintf("not implemented for %s", e.TokenType)
}

// IsNotImplementedErr returns true if given error is a NotImplementedError.
func IsNotImplementedErr(err error) bool {
	return errors.As(err, &NotImplementedError{})
}
