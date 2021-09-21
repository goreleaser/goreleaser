package smtp

import (
	"crypto/tls"
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	gomail "gopkg.in/mail.v2"
)

const (
	defaultSubjectTemplate = `{{ .ProjectName }} {{ .Tag }} is out!`
	defaultBodyTemplate    = `You can view details from: {{ .GitURL }}/releases/tag/{{ .Tag }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "smtp" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.SMTP.Enabled }

type Config struct {
	Host     string `env:"SMTP_HOST,notEmpty"`
	Port     int    `env:"SMTP_PORT,notEmpty"`
	Username string `env:"SMTP_USERNAME,notEmpty"`
	Password string `env:"SMTP_PASSWORD,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.SMTP.SubjectTemplate == "" {
		ctx.Config.Announce.SMTP.SubjectTemplate = defaultSubjectTemplate
	}

	if ctx.Config.Announce.SMTP.BodyTemplate == "" {
		ctx.Config.Announce.SMTP.BodyTemplate = defaultBodyTemplate
	}

	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	subject, err := tmpl.New(ctx).Apply(ctx.Config.Announce.SMTP.SubjectTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to SMTP: %w", err)
	}

	body, err := tmpl.New(ctx).Apply(ctx.Config.Announce.SMTP.BodyTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to SMTP: %w", err)
	}

	m := gomail.NewMessage()

	// Set E-Mail sender
	m.SetHeader("From", ctx.Config.Announce.SMTP.From)

	// Set E-Mail receivers
	receivers := ctx.Config.Announce.SMTP.To
	m.SetHeader("To", receivers...)

	// Set E-Mail subject
	m.SetHeader("Subject", subject)

	// Set E-Mail body. You can set plain text or html with text/html
	m.SetBody("text/plain", body)

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to SMTP: %w", err)
	}

	// Settings for SMTP server
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: ctx.Config.Announce.SMTP.InsecureSkipVerify}

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("announce: failed to announce to SMTP: %w", err)
	}

	log.Infof("announce: The mail has been send from %s to %s\n", ctx.Config.Announce.SMTP.From, receivers)

	return nil
}
