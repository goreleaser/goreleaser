package mattermost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"

	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultColor           = "#2D313E"
	defaultUsername        = `GoReleaser`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
	defaultMessageTitle    = `{{ .ProjectName }} {{ .Tag }} is out!`
)

type Pipe struct{}

func (Pipe) String() string                 { return "mattermost" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Mattermost.Enabled }

type Config struct {
	Webhook string `env:"MATTERMOST_WEBHOOK,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Mattermost.MessageTemplate == "" {
		ctx.Config.Announce.Mattermost.MessageTemplate = defaultMessageTemplate
	}

	if ctx.Config.Announce.Mattermost.TitleTemplate == "" {
		ctx.Config.Announce.Mattermost.TitleTemplate = defaultMessageTitle
	}
	if ctx.Config.Announce.Mattermost.Username == "" {
		ctx.Config.Announce.Mattermost.Username = defaultUsername
	}
	if ctx.Config.Announce.Teams.Color == "" {
		ctx.Config.Announce.Teams.Color = defaultColor
	}

	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Mattermost.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to mattermost: %w", err)
	}

	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Mattermost.TitleTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to teams: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to mattermost: %w", err)
	}

	log.Infof("posting: %q", msg)

	wm := &incomingWebhookRequest{
		Username:    ctx.Config.Announce.Mattermost.Username,
		IconEmoji:   ctx.Config.Announce.Mattermost.IconEmoji,
		IconURL:     ctx.Config.Announce.Mattermost.IconURL,
		ChannelName: ctx.Config.Announce.Mattermost.Channel,
		Attachments: []*mattermostAttachment{
			{
				Title: title,
				Text:  msg,
				Color: ctx.Config.Announce.Teams.Color,
			},
		},
	}

	err = postWebhook(ctx, cfg.Webhook, wm)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to mattermost: %w", err)
	}

	return nil
}

func postWebhook(ctx *context.Context, url string, msg *incomingWebhookRequest) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal the message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to mattermost: %w", err)
	}
	closeBody(r)

	return nil
}

func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}
}

type incomingWebhookRequest struct {
	Text        string                  `json:"text"`
	Username    string                  `json:"username"`
	IconURL     string                  `json:"icon_url"`
	ChannelName string                  `json:"channel"`
	Attachments []*mattermostAttachment `json:"attachments"`
	IconEmoji   string                  `json:"icon_emoji"`
}

type mattermostAttachment struct {
	Fallback   string                       `json:"fallback"`
	Color      string                       `json:"color"`
	Pretext    string                       `json:"pretext"`
	AuthorName string                       `json:"author_name"`
	AuthorLink string                       `json:"author_link"`
	AuthorIcon string                       `json:"author_icon"`
	Title      string                       `json:"title"`
	TitleLink  string                       `json:"title_link"`
	Text       string                       `json:"text"`
	Fields     []*mattermostAttachmentField `json:"fields"`
	Footer     string                       `json:"footer"`
	FooterIcon string                       `json:"footer_icon"`
}

type mattermostAttachmentField struct {
	Title string      `json:"title"`
	Value interface{} `json:"value"`
	Short bool        `json:"short"`
}
