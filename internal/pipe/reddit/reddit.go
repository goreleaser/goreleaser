package reddit

import (
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

const (
	defaultTitleTemplate = `{{ .ProjectName }} {{ .Tag }} is out!`
	defaultURLTemplate   = `{{ .GitURL }}/releases/tag/{{ .Tag }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "reddit" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Reddit.Enabled }

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

func (Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Reddit.TitleTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to reddit: %w", err)
	}

	url, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Reddit.URLTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to reddit: %w", err)
	}

	linkRequest := reddit.SubmitLinkRequest{
		Subreddit: ctx.Config.Announce.Reddit.Sub,
		Title:     title,
		URL:       url,
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to reddit: %w", err)
	}

	credentials := reddit.Credentials{ID: ctx.Config.Announce.Reddit.ApplicationID, Secret: cfg.Secret, Username: ctx.Config.Announce.Reddit.Username, Password: cfg.Password}
	client, err := reddit.NewClient(credentials)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to reddit: %w", err)
	}

	post, _, err := client.Post.SubmitLink(ctx, linkRequest)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to reddit: %w", err)
	}

	log.Infof("announce: The text post is available at: %s\n", post.URL)

	return nil
}
