// Package mastodon announces releases on Mastodon.
package mastodon

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/mattn/go-mastodon"
)

const defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`

type Pipe struct{}

func (Pipe) String() string { return "mastodon" }

func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Mastodon.Enabled)
	return !enable || ctx.Config.Announce.Mastodon.Server == "", err
}

type Config struct {
	ClientID     string `env:"MASTODON_CLIENT_ID,notEmpty"`
	ClientSecret string `env:"MASTODON_CLIENT_SECRET,notEmpty"`
	AccessToken  string `env:"MASTODON_ACCESS_TOKEN,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Mastodon.MessageTemplate == "" {
		ctx.Config.Announce.Mastodon.MessageTemplate = defaultMessageTemplate
	}
	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Mastodon.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	client := mastodon.NewClient(&mastodon.Config{
		Server:       ctx.Config.Announce.Mastodon.Server,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AccessToken:  cfg.AccessToken,
	})

	log.Infof("posting: '%s'", msg)
	if _, err := client.PostStatus(ctx, &mastodon.Toot{
		Status: msg,
	}); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}
