package teams

import (
	"fmt"

	"github.com/apex/log"
	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultColor           = "#2D313E"
	defaultIcon            = "https://goreleaser.com/static/avatar.png"
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
	defaultMessageTitle    = `{{ .ProjectName }} {{ .Tag }} is out!`
)

type Pipe struct{}

func (Pipe) String() string                 { return "teams" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Teams.Enabled }

type Config struct {
	Webhook string `env:"TEAMS_WEBHOOK,notEmpty"`
}

func (p Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Teams.MessageTemplate == "" {
		ctx.Config.Announce.Teams.MessageTemplate = defaultMessageTemplate
	}
	if ctx.Config.Announce.Teams.TitleTemplate == "" {
		ctx.Config.Announce.Teams.TitleTemplate = defaultMessageTitle
	}
	if ctx.Config.Announce.Teams.IconURL == "" {
		ctx.Config.Announce.Teams.IconURL = defaultIcon
	}
	if ctx.Config.Announce.Teams.Color == "" {
		ctx.Config.Announce.Teams.Color = defaultColor
	}
	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Teams.TitleTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Teams.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}

	log.Infof("posting: '%s'", msg)

	client := goteamsnotify.NewClient()
	msgCard := goteamsnotify.NewMessageCard()
	msgCard.Summary = title
	msgCard.ThemeColor = ctx.Config.Announce.Teams.Color

	messageCardSection := goteamsnotify.NewMessageCardSection()
	messageCardSection.ActivityTitle = title
	messageCardSection.ActivityText = msg
	messageCardSection.Markdown = true
	messageCardSection.ActivityImage = ctx.Config.Announce.Teams.IconURL
	err = msgCard.AddSection(messageCardSection)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}
	err = client.Send(cfg.Webhook, msgCard)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}
	return nil
}
