// Package telegram announces releases to Telegram.
package telegram

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"net/http"
	"strconv"
)

const (
	defaultMessageTemplate = `{{ mdv2escape .ProjectName }} {{ mdv2escape .Tag }} is out{{ mdv2escape "!" }} Check it out at {{ mdv2escape .ReleaseURL }}`
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
	err = json.NewEncoder(&b).Encode(args)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.Announce.Telegram.SkipTLSVerify,
	}

	client := &http.Client{
		Transport: customTransport,
	}
	defer client.CloseIdleConnections()

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.ConsumerToken), &b)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	log.Infof("posting: '%s'", args["msg"])
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	defer resp.Body.Close()

	var telegramResponse SendMessageResponse
	err = json.NewDecoder(resp.Body).Decode(&telegramResponse)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	if !telegramResponse.Ok {
		log.Errorf("send telegram failed with %s", telegramResponse.Description)
		return fmt.Errorf("send telegram failed with (%d)%s", telegramResponse.ErrorCode, telegramResponse.Description)
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
	_, err = strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	m["chat_id"] = chatID

	messageThreadIDStr, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Telegram.MessageThreadID)
	if err != nil {
		return nil, fmt.Errorf("telegram: %w", err)
	}
	if messageThreadIDStr != "" {
		messageThreadID, err := strconv.ParseInt(messageThreadIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("telegram: %w", err)
		}
		m["message_thread_id"] = messageThreadID
	}

	return m, nil
}
