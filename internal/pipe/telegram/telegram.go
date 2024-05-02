package telegram

import (
	"fmt"
	"strconv"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultMessageTemplate = `{{ mdv2escape .ProjectName }} {{ mdv2escape .Tag }} is out{{ mdv2escape "!" }} Check it out at {{ mdv2escape .ReleaseURL }}`
	parseModeHTML          = "HTML"
	parseModeMarkdown      = "MarkdownV2"
)

type Pipe struct{}

func (Pipe) String() string                 { return "telegram" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Telegram.Enabled }

type Config struct {
	ConsumerToken string `env:"TELEGRAM_TOKEN,notEmpty"`
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
	msg, chatID, err := getMessageDetails(ctx)
	if err != nil {
		return err
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	log.Infof("posting: '%s'", msg)
	bot, err := api.NewBotAPI(cfg.ConsumerToken)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}

	tm := api.NewMessage(chatID, msg)
	tm.ParseMode = "MarkdownV2"
	_, err = bot.Send(tm)
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	log.Debug("message sent")
	return nil
}

func getMessageDetails(ctx *context.Context) (string, int64, error) {
	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Telegram.MessageTemplate)
	if err != nil {
		return "", 0, fmt.Errorf("telegram: %w", err)
	}
	chatIDStr, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Telegram.ChatID)
	if err != nil {
		return "", 0, fmt.Errorf("telegram: %w", err)
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("telegram: %w", err)
	}

	return msg, chatID, nil
}
