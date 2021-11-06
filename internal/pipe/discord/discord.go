package discord

import (
	"fmt"
	"strconv"

	"github.com/DisgoOrg/disgohook"
	"github.com/DisgoOrg/disgohook/api"
	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultAuthor          = `GoReleaser`
	defaultColor           = "3888754"
	defaultIcon            = "https://goreleaser.com/static/avatar.png"
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "discord" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Discord.Enabled }

type Config struct {
	WebhookID    string `env:"DISCORD_WEBHOOK_ID,notEmpty"`
	WebhookToken string `env:"DISCORD_WEBHOOK_TOKEN,notEmpty"`
}

func (p Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Discord.MessageTemplate == "" {
		ctx.Config.Announce.Discord.MessageTemplate = defaultMessageTemplate
	}
	if ctx.Config.Announce.Discord.IconURL == "" {
		ctx.Config.Announce.Discord.IconURL = defaultIcon
	}
	if ctx.Config.Announce.Discord.Author == "" {
		ctx.Config.Announce.Discord.Author = defaultAuthor
	}
	if ctx.Config.Announce.Discord.Color == "" {
		ctx.Config.Announce.Discord.Color = defaultColor
	}
	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Discord.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to discord: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to discord: %w", err)
	}

	log.Infof("posting: '%s'", msg)

	webhook, err := disgohook.NewWebhookClientByToken(nil, nil, fmt.Sprintf("%s/%s", cfg.WebhookID, cfg.WebhookToken))
	if err != nil {
		return fmt.Errorf("announce: failed to announce to discord: %w", err)
	}
	color, err := strconv.Atoi(ctx.Config.Announce.Discord.Color)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to discord: %w", err)
	}
	if _, err = webhook.SendMessage(api.NewWebhookMessageCreateBuilder().
		AddEmbeds(api.Embed{
			Author: &api.EmbedAuthor{
				Name:    &ctx.Config.Announce.Discord.Author,
				IconURL: &ctx.Config.Announce.Discord.IconURL,
			},
			Description: &msg,
			Color:       &color,
		}).
		Build(),
	); err != nil {
		return fmt.Errorf("announce: failed to announce to discord: %w", err)
	}
	return nil
}
