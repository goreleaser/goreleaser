// Package testctx provides a test context to be used in unit tests.
package testctx

import (
	"time"

	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Opt is an option for a test context.
type Opt func(ctx *context.Context)

func GitHubTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitHub)(ctx)
	WithToken("githubtoken")(ctx)
}

func GitLabTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitLab)(ctx)
	WithToken("gitlabtoken")(ctx)
}

func GiteaTokenType(ctx *context.Context) {
	WithTokenType(context.TokenTypeGitea)(ctx)
	WithToken("giteatoken")(ctx)
}

func WithTokenType(t context.TokenType) Opt {
	return func(ctx *context.Context) {
		ctx.TokenType = t
	}
}

func WithToken(t string) Opt {
	return func(ctx *context.Context) {
		ctx.Token = t
	}
}

func WithVersion(v string) Opt {
	return func(ctx *context.Context) {
		ctx.Version = v
	}
}

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

func WithGitInfo(git context.GitInfo) Opt {
	return func(ctx *context.Context) {
		ctx.Git = git
	}
}

func WithCurrentTag(tag string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.CurrentTag = tag
	}
}

func WithCommit(commig string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.Commit = commig
		ctx.Git.FullCommit = commig
	}
}

func WithCommitDate(d time.Time) Opt {
	return func(ctx *context.Context) {
		ctx.Git.CommitDate = d
	}
}

func WithPreviousTag(tag string) Opt {
	return func(ctx *context.Context) {
		ctx.Git.PreviousTag = tag
	}
}

func WithEnv(env map[string]string) Opt {
	return func(ctx *context.Context) {
		ctx.Env = env
	}
}

func WithDate(t time.Time) Opt {
	return func(ctx *context.Context) {
		ctx.Date = t
	}
}

func WithFakeRuntime(ctx *context.Context) {
	ctx.Runtime = context.Runtime{
		Goos:   "fakeos",
		Goarch: "fakearch",
	}
}

func Skip(keys ...skips.Key) Opt {
	return func(ctx *context.Context) {
		skips.Set(ctx, keys...)
	}
}

func Snapshot(ctx *context.Context) {
	ctx.Snapshot = true
}

func NewWithCfg(c config.Project, opts ...Opt) *context.Context {
	ctx := context.New(c)
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

func New(opts ...Opt) *context.Context {
	return NewWithCfg(config.Project{}, opts...)
}
