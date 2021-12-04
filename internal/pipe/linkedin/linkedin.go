package linkedin

import (
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`

type Pipe struct{}

func (Pipe) String() string                 { return "linkedin" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.LinkedIn.Enabled }

type Config struct {
	AccessToken string `env:"LINKEDIN_ACCESS_TOKEN,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.LinkedIn.MessageTemplate == "" {
		ctx.Config.Announce.LinkedIn.MessageTemplate = defaultMessageTemplate
	}

	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	message, err := tmpl.New(ctx).Apply(ctx.Config.Announce.LinkedIn.MessageTemplate)
	if err != nil {
		return fmt.Errorf("failed to announce to linkedin: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to announce to linkedin: %w", err)
	}

	c, err := createLinkedInClient(oauthClientConfig{
		Context:     ctx,
		AccessToken: cfg.AccessToken,
	})
	if err != nil {
		return fmt.Errorf("failed to announce to linkedin: %w", err)
	}

	url, err := c.Share(message)
	if err != nil {
		return fmt.Errorf("failed to announce to linkedin: %w", err)
	}

	log.Infof("The text post is available at: %s\n", url)

	return nil
}
