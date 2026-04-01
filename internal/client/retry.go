package client

import (
	"errors"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var _ FullClient = &retryableClient{}

type retryableClient struct {
	c FullClient
}

// CreateFiles implements [FullClient].
func (r *retryableClient) CreateFiles(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, message string, files []RepoFile) (err error) {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.CreateFiles(ctx, commitAuthor, repo, message, files)
		},
		isRetriable,
	)
}

// GenerateReleaseNotes implements [FullClient].
func (r *retryableClient) GenerateReleaseNotes(ctx *context.Context, repo Repo, prev string, current string) (string, error) {
	return retryx.DoWithData(
		ctx.Config.Retry,
		func() (string, error) {
			return r.c.GenerateReleaseNotes(ctx, repo, prev, current)
		},
		isRetriable,
	)
}

// OpenPullRequest implements [FullClient].
func (r *retryableClient) OpenPullRequest(ctx *context.Context, base Repo, head Repo, title string, draft bool) error {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.OpenPullRequest(ctx, base, head, title, draft)
		},
		isRetriable,
	)
}

// SyncFork implements [FullClient].
func (r *retryableClient) SyncFork(ctx *context.Context, head Repo, base Repo) error {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.SyncFork(ctx, head, base)
		},
		isRetriable,
	)
}

// Changelog implements [Client].
func (r *retryableClient) Changelog(ctx *context.Context, repo Repo, prev string, current string) ([]ChangelogItem, error) {
	return retryx.DoWithData(
		ctx.Config.Retry,
		func() ([]ChangelogItem, error) {
			return r.c.Changelog(ctx, repo, prev, current)
		},
		isRetriable,
	)
}

// CloseMilestone implements [Client].
func (r *retryableClient) CloseMilestone(ctx *context.Context, repo Repo, title string) (err error) {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.CloseMilestone(ctx, repo, title)
		},
		isRetriable,
	)
}

// CreateFile implements [Client].
func (r *retryableClient) CreateFile(ctx *context.Context, commitAuthor config.CommitAuthor, repo Repo, content []byte, path string, message string) (err error) {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.CreateFile(ctx, commitAuthor, repo, content, path, message)
		},
		isRetriable,
	)
}

// CreateRelease implements [Client].
func (r *retryableClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	return retryx.DoWithData(
		ctx.Config.Retry,
		func() (string, error) {
			return r.c.CreateRelease(ctx, body)
		},
		isRetriable,
	)
}

// PublishRelease implements [Client].
func (r *retryableClient) PublishRelease(ctx *context.Context, releaseID string) (err error) {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.PublishRelease(ctx, releaseID)
		},
		isRetriable,
	)
}

// ReleaseURLTemplate implements [Client].
func (r *retryableClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	return r.c.ReleaseURLTemplate(ctx)
}

// Upload implements [Client].
func (r *retryableClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact) (err error) {
	return retryx.Do(
		ctx.Config.Retry,
		func() error {
			return r.c.Upload(ctx, releaseID, artifact)
		},
		isRetriable,
	)
}

func isRetriable(err error) bool {
	_, ok := errors.AsType[RetriableError](err)
	return ok
}
