package smtp

import (
	"crypto/tls"
	"fmt"

	"github.com/caarlos0/env/v9"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	gomail "gopkg.in/mail.v2"
)

const (
	defaultSubjectTemplate = `{{ .ProjectName }} {{ .Tag }} is out!`
	defaultBodyTemplate    = `You can view details from: {{ .ReleaseURL }}`
)

type Pipe struct{}

func (Pipe) String() string                 { return "smtp" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.SMTP.Enabled }

type Config struct {
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT"`
	Username string `env:"SMTP_USERNAME"`
	Password string `env:"SMTP_PASSWORD,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.SMTP.BodyTemplate == "" {
		ctx.Config.Announce.SMTP.BodyTemplate = defaultBodyTemplate
	}

	if ctx.Config.Announce.SMTP.SubjectTemplate == "" {
		ctx.Config.Announce.SMTP.SubjectTemplate = defaultSubjectTemplate
	}

	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	subject, err := tmpl.New(ctx).Apply(ctx.Config.Announce.SMTP.SubjectTemplate)
	if err != nil {
		return fmt.Errorf("SMTP: %w", err)
	}

	body, err := tmpl.New(ctx).Apply(ctx.Config.Announce.SMTP.BodyTemplate)
	if err != nil {
		return fmt.Errorf("SMTP: %w", err)
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

	cfg, err := getConfig(ctx.Config.Announce.SMTP)
	if err != nil {
		return err
	}

	// Settings for SMTP server
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: ctx.Config.Announce.SMTP.InsecureSkipVerify}

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("SMTP: %w", err)
	}

	log.Infof("The mail has been send from %s to %s\n", ctx.Config.Announce.SMTP.From, receivers)

	return nil
}

var (
	errNoPort     = fmt.Errorf("SMTP: missing smtp.port or $SMTP_PORT")
	errNoUsername = fmt.Errorf("SMTP: missing smtp.username or $SMTP_USERNAME")
	errNoHost     = fmt.Errorf("SMTP: missing smtp.host or $SMTP_HOST")
)

func getConfig(smtp config.SMTP) (Config, error) {
	cfg := Config{
		Host:     smtp.Host,
		Port:     smtp.Port,
		Username: smtp.Username,
	}
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("SMTP: %w", err)
	}
	if cfg.Username == "" {
		return cfg, errNoUsername
	}
	if cfg.Host == "" {
		return cfg, errNoHost
	}
	if cfg.Port == 0 {
		return cfg, errNoPort
	}
	return cfg, nil
}
