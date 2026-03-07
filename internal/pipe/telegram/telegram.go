// Package telegram announces releases to Telegram.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultMessageTemplate = `{{ print .ProjectName " " .Tag " is out! Check it out at " .ReleaseURL | mdv2escape }}`
	parseModeHTML          = "HTML"
	parseModeMarkdown      = "MarkdownV2"
)

type Pipe struct{}

func (Pipe) String() string { return "telegram" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Telegram.Enabled)
	return !enable, err
}

type Config struct {
	ConsumerToken string `env:"TELEGRAM_TOKEN,notEmpty"`
}

type SendMessageResponse struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Telegram.MessageTemplate == "" {
		ctx.Config.Announce.Telegram.MessageTemplate = defaultMessageTemplate
	}
	switch ctx.Config.Announce.Telegram.ParseMode {
	case parseModeHTML, parseModeMarkdown:
		break
	default:
		ctx.Config.Announce.Telegram.ParseMode = parseModeMarkdown
	}
	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	args, err := getMessageDetails(ctx)
	if err != nil {
		return err
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(args); err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.ConsumerToken), &b)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	log.Infof("posting: '%s'", args["text"])
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: status code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var telegramResponse SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResponse); err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	if !telegramResponse.Ok {
		return fmt.Errorf("telegram: send failed with error code %d: %s", telegramResponse.ErrorCode, telegramResponse.Description)
	}

	log.Debug("message sent")
	return nil
}

func getMessageDetails(ctx *context.Context) (map[string]any, error) {
	m := map[string]any{}
	if ctx.Config.Announce.Telegram.ParseMode != "" {
		m["parse_mode"] = ctx.Config.Announce.Telegram.ParseMode
	}
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Telegram.MessageTemplate)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	m["text"] = msg

	chatID, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Telegram.ChatID)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	m["chat_id"] = chatID

	return m, nil
}
