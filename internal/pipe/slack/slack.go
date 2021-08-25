package slack

import (
	"fmt"
	"github.com/slack-go/slack"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultUsername        = `GoReleaser`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
)

type Pipe struct{}

func (Pipe) String() string { return "slack" }

type Config struct {
	Webhook string `env:"SLACK_WEBHOOK,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Slack.MessageTemplate == "" {
		ctx.Config.Announce.Slack.MessageTemplate = defaultMessageTemplate
	}
	if ctx.Config.Announce.Slack.Username == "" {
		ctx.Config.Announce.Slack.Username = defaultUsername
	}
	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	if ctx.SkipAnnounce {
		return pipe.ErrSkipAnnounceEnabled
	}
	if !ctx.Config.Announce.Slack.Enabled {
		return pipe.ErrSkipDisabledPipe
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Slack.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	log.Infof("posting: '%s'", msg)

	wm := &slack.WebhookMessage{
		Username:  ctx.Config.Announce.Slack.Username,
		IconEmoji: ctx.Config.Announce.Slack.IconEmoji,
		IconURL:   ctx.Config.Announce.Slack.IconURL,
		Channel:   ctx.Config.Announce.Slack.Channel,
		Text:      msg,
	}

	err = slack.PostWebhook(cfg.Webhook, wm)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	return nil
}
