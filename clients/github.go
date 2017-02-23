package clients

import (
	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/oauth2"
)

// GitHub client for the given token
func GitHub(ctx *context.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ctx.Token},
	)
	return github.NewClient(oauth2.NewClient(ctx, ts))
}
