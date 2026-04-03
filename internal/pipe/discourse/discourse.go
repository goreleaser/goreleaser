// Package discourse announces to a Discourse forum.
package discourse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
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
		return err
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Discourse.MessageTemplate)
	if err != nil {
		return err
	}

	// Make 'server' a required config field.
	if ctx.Config.Announce.Discourse.Server == "" {
		return errors.New("'server' is a required config key")
	}

	// Make 'category_id' a required config field.
	if ctx.Config.Announce.Discourse.CategoryID == 0 {
		return errors.New("'category_id' is a required config key")
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return err
	}

	pr := &postsRequest{
		Title:    title,
		Raw:      msg,
		Category: ctx.Config.Announce.Discourse.CategoryID,
	}

	endpoint := ctx.Config.Announce.Discourse.Server + "/posts.json"

	payload, err := json.Marshal(pr)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "goreleaser/v2")
	req.Header.Set("Api-Username", ctx.Config.Announce.Discourse.Username)
	req.Header.Set("Api-Key", cfg.APIKey)

	var statusCode int
	return retryx.Do(ctx.Config.Retry, func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			statusCode = 0
			return err
		}
		defer resp.Body.Close()
		statusCode = resp.StatusCode

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("error posting to Discourse, check your config again, HTTP code: %d", resp.StatusCode)
		}

		return nil
	}, func(err error) bool {
		return retryx.IsRetriableHTTPError(statusCode, err)
	})
}
