package testctx

import (
	"time"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

type Opt func(ctx *context.Context)

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

func SkipPublish(ctx *context.Context) {
	ctx.SkipPublish = true
}

func SkipAnnounce(ctx *context.Context) {
	ctx.SkipAnnounce = true
}

func SkipDocker(ctx *context.Context) {
	ctx.SkipDocker = true
}

func SkipValidate(ctx *context.Context) {
	ctx.SkipValidate = true
}

func Snapshot(ctx *context.Context) {
	ctx.Snapshot = true
}

func SkipSign(ctx *context.Context) {
	ctx.SkipSign = true
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
