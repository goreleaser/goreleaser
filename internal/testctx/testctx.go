package testctx

import (
	"time"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Opt func(ctx *context.Context)

func WithConfig(c config.Project) Opt {
	return func(ctx *context.Context) {
		ctx.Config = c
	}
}

func WithTokenType(t context.TokenType) Opt {
	return func(ctx *context.Context) {
		ctx.TokenType = t
	}
}

func WithVersion(v string) Opt {
	return func(ctx *context.Context) {
		ctx.Version = v
	}
}

func WithSemver(v context.Semver) Opt {
	return func(ctx *context.Context) {
		ctx.Semver = v
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

func SkipPublish(ctx *context.Context) {
	ctx.SkipPublish = true
}

func Snapshot(ctx *context.Context) {
	ctx.Snapshot = true
}

func SkipSign(ctx *context.Context) {
	ctx.SkipSign = true
}

func NewWithCfg(c config.Project, opts ...Opt) *context.Context {
	return New(append(opts, WithConfig(c))...)
}

func New(opts ...Opt) *context.Context {
	ctx := context.New(config.Project{})
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}
