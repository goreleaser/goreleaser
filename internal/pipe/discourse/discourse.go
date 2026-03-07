// Package discourse announces to a Discourse forum.
package discourse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultTitleTemplate   = `{{ .ProjectName }} {{ .Tag }} is out!`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
	defaultUsername        = "system"
)

type Pipe struct{}

func (Pipe) String() string { return "discourse" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Discourse.Enabled)
	return !enable, err
}

type Config struct {
	APIKey string `env:"DISCOURSE_API_KEY,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Discourse.TitleTemplate == "" {
		ctx.Config.Announce.Discourse.TitleTemplate = defaultTitleTemplate
	}

	if ctx.Config.Announce.Discourse.MessageTemplate == "" {
		ctx.Config.Announce.Discourse.MessageTemplate = defaultMessageTemplate
	}

	if ctx.Config.Announce.Discourse.Username == "" {
		ctx.Config.Announce.Discourse.Username = defaultUsername
	}

	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Discourse.TitleTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Discourse.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	// Make 'server' a required config field.
	if ctx.Config.Announce.Discourse.Server == "" {
		return fmt.Errorf("%s: 'server' is a required config key", p)
	}

	// Make 'category_id' a required config field.
	if ctx.Config.Announce.Discourse.CategoryID == 0 {
		return fmt.Errorf("%s: 'category_id' is a required config key", p)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	pr := &postsRequest{
		Title:    title,
		Raw:      msg,
		Category: ctx.Config.Announce.Discourse.CategoryID,
	}

	endpoint := ctx.Config.Announce.Discourse.Server + "/posts.json"

	payload, err := json.Marshal(pr)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "goreleaser/v2")
	req.Header.Set("Api-Username", ctx.Config.Announce.Discourse.Username)
	req.Header.Set("Api-Key", cfg.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s: There was an error posting to Discourse. Check your config again. HTTP code: %d", p, resp.StatusCode)
	}

	return nil
}
