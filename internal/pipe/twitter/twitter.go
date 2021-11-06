package twitter

import (
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`

type Pipe struct{}

func (Pipe) String() string                 { return "twitter" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Twitter.Enabled }

type Config struct {
	ConsumerKey    string `env:"TWITTER_CONSUMER_KEY,notEmpty"`
	ConsumerSecret string `env:"TWITTER_CONSUMER_SECRET,notEmpty"`
	AccessToken    string `env:"TWITTER_ACCESS_TOKEN,notEmpty"`
	AccessSecret   string `env:"TWITTER_ACCESS_TOKEN_SECRET,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Twitter.MessageTemplate == "" {
		ctx.Config.Announce.Twitter.MessageTemplate = defaultMessageTemplate
	}
	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Twitter.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to twitter: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to twitter: %w", err)
	}

	log.Infof("posting: '%s'", msg)
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	client := twitter.NewClient(config.Client(oauth1.NoContext, token))
	if _, _, err := client.Statuses.Update(msg, nil); err != nil {
		return fmt.Errorf("announce: failed to announce to twitter: %w", err)
	}
	return nil
}
