// Package client contains the client implementations for several providers.
package client

import (
	"fmt"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	// maxReleaseBodyLength defines the max characters size of the body
	maxReleaseBodyLength = 125000
	// ellipsis to be used when release notes body is too long
	ellipsis = "..."
)

// ErrNotImplemented is returned when a client does not implement certain feature.
var ErrNotImplemented = fmt.Errorf("not implemented")

// ErrReleaseDisabled happens when a configuration tries to use the default
// url_template even though the release is disabled.
var ErrReleaseDisabled = fmt.Errorf("release is disabled, cannot use default url_template")

// Info of the repository.
type Info struct {
	Description string
	Homepage    string
	URL         string
}

type Repo struct {
	Owner         string
	Name          string
	Branch        string
	GitURL        string
	GitSSHCommand string
	PrivateKey    string
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
	// Creates a release. It's marked as draft if possible (should call PublishRelease to finish publishing).
	CreateRelease(ctx *context.Context, body string) (releaseID string, err error)
	PublishRelease(ctx *context.Context, releaseID string) (err error)
	Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact, file *os.File) (err error)
	Changelog(ctx *context.Context, repo Repo, prev, current string) ([]ChangelogItem, error)
	ReleaseURLTemplater
	FileCreator
}

// ChangelogItem represents a changelog item, basically, a commit and its author.
type ChangelogItem struct {
	SHA            string
	Message        string
	AuthorName     string
	AuthorEmail    string
	AuthorUsername string
}

// ReleaseURLTemplater provides the release URL as a template, containing the
// artifact name as well.
type ReleaseURLTemplater interface {
	ReleaseURLTemplate(ctx *context.Context) (string, error)
}

// RepoFile is a file to be created.
type RepoFile struct {
	Content    []byte
	Path       string
	Identifier string // for the use of the caller.
}

// FileCreator can create the given file to some code repository.
type FileCreator interface {
	CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path, message string) (err error)
}

// FilesCreator can create the multiple files in some repository and in a single commit.
type FilesCreator interface {
	FileCreator
	CreateFiles(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, message string, files []RepoFile) (err error)
}

// ReleaseNotesGenerator can generate release notes.
type ReleaseNotesGenerator interface {
	GenerateReleaseNotes(ctx *context.Context, repo Repo, prev, current string) (string, error)
}

// ForkSyncer can sync forks.
type ForkSyncer interface {
	SyncFork(ctx *context.Context, head, base Repo) error
}

// PullRequestOpener can open pull requests.
type PullRequestOpener interface {
	OpenPullRequest(ctx *context.Context, base, head Repo, title string, draft bool) error
}

// New creates a new client depending on the token type.
func New(ctx *context.Context) (Client, error) {
	return newWithToken(ctx, ctx.Token)
}

// NewReleaseClient returns a ReleaserURLTemplater, handling the possibility of
// the release being disabled.
func NewReleaseClient(ctx *context.Context) (ReleaseURLTemplater, error) {
	disable, err := tmpl.New(ctx).Bool(ctx.Config.Release.Disable)
	if err != nil {
		return nil, err
	}
	if disable {
		return errURLTemplater{}, nil
	}
	return New(ctx)
}

var _ ReleaseURLTemplater = errURLTemplater{}

type errURLTemplater struct{}

func (errURLTemplater) ReleaseURLTemplate(_ *context.Context) (string, error) {
	return "", ErrReleaseDisabled
}

func newWithToken(ctx *context.Context, token string) (Client, error) {
	log.WithField("type", ctx.TokenType).Debug("token type")
	switch ctx.TokenType {
	case context.TokenTypeGitHub:
		return newGitHub(ctx, token)
	case context.TokenTypeGitLab:
		return newGitLab(ctx, token)
	case context.TokenTypeGitea:
		return newGitea(ctx, token)
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

func truncateReleaseBody(body string) string {
	if len(body) > maxReleaseBodyLength {
		body = body[1:(maxReleaseBodyLength-len(ellipsis))] + ellipsis
	}
	return body
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
