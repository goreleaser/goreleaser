// Package testctx provides a test context to be used in unit tests.
package testctx

import (
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Opt is an option for a test context.
type Opt func(ctx *context.Context)

// GitHubTokenType configures the context to use a GitHub token.
func GitHubTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitHub)(ctx)
	WithToken("githubtoken")(ctx)
}

// GitLabTokenType configures the context to use a GitLab token.
func GitLabTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitLab)(ctx)
	WithToken("gitlabtoken")(ctx)
}

// GiteaTokenType configures the context to use a Gitea token.
func GiteaTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitea)(ctx)
	WithToken("giteatoken")(ctx)
}

// WithTokenType sets the token type.
func WithTokenType(t context.TokenType) Opt {
	return func(ctx *context.Context) {
		ctx.TokenType = t
	}
}

// WithToken sets the token.
func WithToken(t string) Opt {
	return func(ctx *context.Context) {
		ctx.Token = t
	}
}

// WithVersion sets the version.
func WithVersion(v string) Opt {
	return func(ctx *context.Context) {
		ctx.Version = v
	}
}

// WithSemver sets the semver.
func WithSemver(major, minor, patch uint64, prerelease string) Opt {
	return func(ctx *context.Context) {
		ctx.Semver = context.Semver{
			Major:      major,
			Minor:      minor,
			Patch:      patch,
			Prerelease: prerelease,
		}
	}
}

// WithGitInfo sets the git info.
func WithGitInfo(git context.GitInfo) Opt {
	return func(ctx *context.Context) {
		ctx.Git = git
	}
}

// WithCurrentTag sets the current tag.
func WithCurrentTag(tag string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.CurrentTag = tag
	}
}

// WithCommit sets the commit.
func WithCommit(commit string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.Commit = commit
		ctx.Git.FullCommit = commit
	}
}

// WithCommitDate sets the commit date.
func WithCommitDate(d time.Time) Opt {
	return func(ctx *context.Context) {
		ctx.Git.CommitDate = d
	}
}

// WithPreviousTag sets the previous tag.
func WithPreviousTag(tag string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.PreviousTag = tag
	}
}

// WithEnv sets the env.
func WithEnv(env map[string]string) Opt {
	return func(ctx *context.Context) {
		ctx.Env = env
	}
}

// WithDate sets the date.
func WithDate(t time.Time) Opt {
	return func(ctx *context.Context) {
		ctx.Date = t
	}
}

// WithFakeRuntime sets the runtime to fake values.
func WithFakeRuntime(ctx *context.Context) {
	ctx.Runtime = context.Runtime{
		Goos:   "fakeos",
		Goarch: "fakearch",
	}
}

// Skip skips the given keys.
func Skip(keys ...skips.Key) Opt {
	return func(ctx *context.Context) {
		skips.Set(ctx, keys...)
	}
}

// Snapshot sets the snapshot flag.
func Snapshot(ctx *context.Context) {
	ctx.Snapshot = true
}

// Partial sets the partial flag.
func Partial(ctx *context.Context) {
	ctx.Partial = true
}

// NewWithCfg new context with the given config.
func NewWithCfg(c config.Project, opts ...Opt) *context.Context {
	ctx := context.New(c)
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

// New new context.
func New(opts ...Opt) *context.Context {
	return NewWithCfg(config.Project{}, opts...)
}
