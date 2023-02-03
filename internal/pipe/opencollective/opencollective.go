package opencollective

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v6"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultTitleTemplate   = `{{ .Tag }}`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out! Check it out at <a href="{{ .ReleaseURL }}">{{ .ReleaseURL }}</a>`
	endpoint               = "https://api.opencollective.com/graphql/v2"
)

type Pipe struct{}

func (Pipe) String() string { return "opencollective" }

func (Pipe) Skip(ctx *context.Context) bool {
	return !ctx.Config.Announce.OpenCollective.Enabled || ctx.Config.Announce.OpenCollective.Slug == ""
}

type Config struct {
	Token string `env:"OPENCOLLECTIVE_TOKEN,notEmpty"`
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.OpenCollective.TitleTemplate == "" {
		ctx.Config.Announce.OpenCollective.TitleTemplate = defaultTitleTemplate
	}
	if ctx.Config.Announce.OpenCollective.MessageTemplate == "" {
		ctx.Config.Announce.OpenCollective.MessageTemplate = defaultMessageTemplate
	}
	return nil
}

func (Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.OpenCollective.TitleTemplate)
	if err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}
	html, err := tmpl.New(ctx).Apply(ctx.Config.Announce.OpenCollective.MessageTemplate)
	if err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}

	log.Infof("posting: %q | %q", title, html)

	id, err := createUpdate(ctx, title, html, ctx.Config.Announce.OpenCollective.Slug, cfg.Token)
	if err != nil {
		return err
	}

	return publishUpdate(ctx, id, cfg.Token)
}

func createUpdate(ctx *context.Context, title, html, slug, token string) (string, error) {
	mutation := `mutation (
  $update: UpdateCreateInput!
) {
  createUpdate(update: $update) {
    id
  }
}`
	payload, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}{
		Query: mutation,
		Variables: map[string]any{
			"title": title,
			"html":  html,
			"account": map[string]any{
				"slug": slug,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("could not marshal opencollective mutation: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Personal-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("incorrect response from opencollective: %s", resp.Status)
	}

	var envelope struct {
		Data struct {
			CreateUpdate struct {
				ID string `json:"id"`
			} `json:"createUpdate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}

	return envelope.Data.CreateUpdate.ID, nil
}

func publishUpdate(ctx *context.Context, id, token string) error {
	mutation := `mutation (
  $id: String!
  $audience: UpdateAudience
) {
  publishUpdate(id: $id, notificationAudience: $audience) {
    id
  }
}`
	payload, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}{
		Query: mutation,
		Variables: map[string]any{
			"id":       id,
			"audience": "ALL",
		},
	})
	if err != nil {
		return fmt.Errorf("could not marshal opencollective mutation: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Personal-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("incorrect response from opencollective: %s", resp.Status)
	}

	return nil
}
