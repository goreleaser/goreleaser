// Package reddit announces the release on Reddit.
package reddit

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/go-reddit/v3/reddit"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultTitleTemplate = `{{ .ProjectName }} {{ .Tag }} is out!`
	defaultURLTemplate   = `{{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string { return "reddit" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Reddit.Enabled)
	return !enable, err
}

type Config struct {
	Secret   string `env:"REDDIT_SECRET,notEmpty"`
	Password string `env:"REDDIT_PASSWORD,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Reddit.TitleTemplate == "" {
		ctx.Config.Announce.Reddit.TitleTemplate = defaultTitleTemplate
	}

	if ctx.Config.Announce.Reddit.URLTemplate == "" {
		ctx.Config.Announce.Reddit.URLTemplate = defaultURLTemplate
	}

	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Reddit.TitleTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	url, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Reddit.URLTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	linkRequest := reddit.SubmitLinkRequest{
		Subreddit: ctx.Config.Announce.Reddit.Sub,
		Title:     title,
		URL:       url,
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	credentials := reddit.Credentials{ID: ctx.Config.Announce.Reddit.ApplicationID, Secret: cfg.Secret, Username: ctx.Config.Announce.Reddit.Username, Password: cfg.Password}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	post, _, err := client.Post.SubmitLink(ctx, linkRequest)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	log.Infof("The text post is available at: %s\n", post.URL)

	return nil
}
