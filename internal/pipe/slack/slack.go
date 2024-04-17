package slack

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
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
		return fmt.Errorf("slack: %w", err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("slack: %w", err)
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
		return fmt.Errorf("slack: %w", err)
	}

	return nil
}

func parseAdvancedFormatting(ctx *context.Context) (*slack.Blocks, []slack.Attachment, error) {
	var blocks *slack.Blocks
	if in := ctx.Config.Announce.Slack.Blocks; len(in) > 0 {
		blocks = &slack.Blocks{BlockSet: make([]slack.Block, 0, len(in))}

		if err := unmarshal(ctx, in, blocks); err != nil {
			return nil, nil, fmt.Errorf("slack blocks: %w", err)
		}
	}

	var attachments []slack.Attachment
	if in := ctx.Config.Announce.Slack.Attachments; len(in) > 0 {
		attachments = make([]slack.Attachment, 0, len(in))

		if err := unmarshal(ctx, in, &attachments); err != nil {
			return nil, nil, fmt.Errorf("slack attachments: %w", err)
		}
	}

	return blocks, attachments, nil
}

func unmarshal(ctx *context.Context, in interface{}, target interface{}) error {
	jazon, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal input as JSON: %w", err)
	}

	body := string(jazon)
	// ensure that double quotes that are inside the string get un-escaped so they can be interpreted for templates
	body = strings.ReplaceAll(body, "\\\"", "\"")

	tplApplied, err := tmpl.New(ctx).Apply(body)
	if err != nil {
		return fmt.Errorf("failed to evaluate template: %w", err)
	}

	if err = json.Unmarshal([]byte(tplApplied), target); err != nil {
		return fmt.Errorf("failed to unmarshal into target: %w", err)
	}

	return nil
}
