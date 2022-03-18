package slack

import (
	"encoding/json"
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/slack-go/slack"
)

const (
	defaultUsername        = `GoReleaser`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "slack" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Slack.Enabled }

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
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Slack.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	log.Infof("posting: '%s'", msg)

	// optional processing of advanced formatting options
	blocks, attachments, err := parseAdvancedFormatting(ctx)
	if err != nil {
		return err
	}

	wm := &slack.WebhookMessage{
		Username:  ctx.Config.Announce.Slack.Username,
		IconEmoji: ctx.Config.Announce.Slack.IconEmoji,
		IconURL:   ctx.Config.Announce.Slack.IconURL,
		Channel:   ctx.Config.Announce.Slack.Channel,
		Text:      msg,

		// optional enrichments
		Blocks:      blocks,
		Attachments: attachments,
	}

	err = slack.PostWebhook(cfg.Webhook, wm)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to slack: %w", err)
	}

	return nil
}

func parseAdvancedFormatting(ctx *context.Context) (*slack.Blocks, []slack.Attachment, error) {
	var blocks *slack.Blocks
	if in := ctx.Config.Announce.Slack.Blocks; len(in) > 0 {
		blocks = &slack.Blocks{BlockSet: make([]slack.Block, 0, len(in))}

		if err := unmarshal(ctx, in, blocks); err != nil {
			return nil, nil, fmt.Errorf("announce: slack blocks: %w", err)
		}
	}

	var attachments []slack.Attachment
	if in := ctx.Config.Announce.Slack.Attachments; len(in) > 0 {
		attachments = make([]slack.Attachment, 0, len(in))

		if err := unmarshal(ctx, in, &attachments); err != nil {
			return nil, nil, fmt.Errorf("announce: slack attachments: %w", err)
		}
	}

	return blocks, attachments, nil
}

func unmarshal(ctx *context.Context, in interface{}, target interface{}) error {
	jazon, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("announce: failed to marshal input as JSON: %w", err)
	}

	tplApplied, err := tmpl.New(ctx).Apply(string(jazon))
	if err != nil {
		return fmt.Errorf("announce: failed to evaluate template: %w", err)
	}

	if err = json.Unmarshal([]byte(tplApplied), target); err != nil {
		return fmt.Errorf("announce: failed to unmarshal into target: %w", err)
	}

	return nil
}
