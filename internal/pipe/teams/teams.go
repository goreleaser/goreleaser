// Package teams announces new releases to Microsoft Teams.
package teams

import (
	"fmt"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultColor           = "#2D313E"
	defaultIcon            = "https://goreleaser.com/static/avatar.png"
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
	defaultMessageTitle    = `{{ .ProjectName }} {{ .Tag }} is out!`
)

type Pipe struct{}

func (Pipe) String() string { return "teams" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Teams.Enabled)
	return !enable, err
}

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
		return fmt.Errorf("%s: %w", p, err)
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Teams.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	log.Infof("posting: '%s'", msg)

	client := goteamsnotify.NewTeamsClient()
	msgCard := messagecard.NewMessageCard()
	msgCard.Summary = title
	msgCard.ThemeColor = ctx.Config.Announce.Teams.Color

	messageCardSection := messagecard.NewSection()
	messageCardSection.ActivityTitle = title
	messageCardSection.ActivityText = msg
	messageCardSection.Markdown = true
	messageCardSection.ActivityImage = ctx.Config.Announce.Teams.IconURL
	err = msgCard.AddSection(messageCardSection)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	err = client.Send(cfg.Webhook, msgCard)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}
