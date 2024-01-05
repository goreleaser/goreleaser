package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/caarlos0/env/v9"
	"github.com/caarlos0/log"
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
		return fmt.Errorf("discord: %w", err)
	}

	var cfg Config
	if err = env.Parse(&cfg); err != nil {
		return fmt.Errorf("discord: %w", err)
	}

	log.Infof("posting: '%s'", msg)

	color, err := strconv.Atoi(ctx.Config.Announce.Discord.Color)
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}

	u, err := url.Parse("https://discord.com/api")
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}
	u = u.JoinPath("webhooks", cfg.WebhookID, cfg.WebhookToken)

	bts, err := json.Marshal(WebhookMessageCreate{
		Embeds: []Embed{
			{
				Author: &EmbedAuthor{
					Name:    ctx.Config.Announce.Discord.Author,
					IconURL: ctx.Config.Announce.Discord.IconURL,
				},
				Description: msg,
				Color:       color,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewReader(bts))
	if err != nil {
		return fmt.Errorf("discord: %w", err)
	}

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("discord: %s", resp.Status)
	}

	return nil
}

type WebhookMessageCreate struct {
	Embeds []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}
