// Package twitter announces releases on Twitter.
package twitter

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`

type Pipe struct{}

func (Pipe) String() string { return "twitter" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Twitter.Enabled)
	return !enable, err
}

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

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Twitter.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	log.Infof("posting: '%s'", msg)
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	client := twitter.NewClient(config.Client(oauth1.NoContext, token))
	if _, _, err := client.Statuses.Update(msg, nil); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}
