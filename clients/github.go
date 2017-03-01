package clients

import (
	"errors"

	"github.com/google/go-github/github"
	"github.com/goreleaser/goreleaser/context"
	"golang.org/x/oauth2"
)

// ErrMissingToken indicates an error when GITHUB_TOKEN is missing in the environment
var ErrMissingToken = errors.New("Missing GITHUB_TOKEN")

// GitHub client for the given token
func GitHub(ctx *context.Context) (*github.Client, error) {
	if ctx.Token == "" {
		return nil, ErrMissingToken
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ctx.Token},
	)
	return github.NewClient(oauth2.NewClient(ctx, ts)), nil
}
