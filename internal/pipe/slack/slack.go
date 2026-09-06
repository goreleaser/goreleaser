// Package slack announces releases to Slack.
package slack

import (
	"encoding/json"
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/slack-go/slack"
)

const (
	defaultUsername        = `GoReleaser`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string { return "slack" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Slack.Enabled)
	return !enable, err
}

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

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Slack.MessageTemplate)
	if err != nil {
		return err
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return err
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

	return retryx.Do(ctx, ctx.Config.Retry, func() error {
		return slack.PostWebhookContext(ctx, cfg.Webhook, wm)
	}, retryx.IsNetworkError)
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

func unmarshal(ctx *context.Context, in any, target any) error {
	jazon, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal input as JSON: %w", err)
	}

	var raw any
	if err := json.Unmarshal(jazon, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal input as JSON: %w", err)
	}

	tplApplied, err := applyTemplates(tmpl.New(ctx), raw)
	if err != nil {
		return fmt.Errorf("failed to evaluate template: %w", err)
	}

	jazon, err = json.Marshal(tplApplied)
	if err != nil {
		return fmt.Errorf("failed to marshal rendered input as JSON: %w", err)
	}

	if err = json.Unmarshal(jazon, target); err != nil {
		return fmt.Errorf("failed to unmarshal into target: %w", err)
	}

	return nil
}

func applyTemplates(tpl *tmpl.Template, in any) (any, error) {
	switch v := in.(type) {
	case string:
		return tpl.Apply(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			applied, err := applyTemplates(tpl, item)
			if err != nil {
				return nil, err
			}
			out[i] = applied
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			applied, err := applyTemplates(tpl, value)
			if err != nil {
				return nil, err
			}
			out[key] = applied
		}
		return out, nil
	default:
		return v, nil
	}
}
