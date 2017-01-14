package clients

import (
	"context"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func Github(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}
