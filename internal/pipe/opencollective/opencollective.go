package opencollective

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultTitleTemplate   = `{{ .Tag }}`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out!<br/>Check it out at <a href="{{ .ReleaseURL }}">{{ .ReleaseURL }}</a>`
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

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}

	log.Infof("posting: %q | %q", title, html)

	id, err := createUpdate(ctx, title, html, ctx.Config.Announce.OpenCollective.Slug, cfg.Token)
	if err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}

	if err := publishUpdate(ctx, id, cfg.Token); err != nil {
		return fmt.Errorf("opencollective: %w", err)
	}

	return nil
}

type payload struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func createUpdate(ctx *context.Context, title, html, slug, token string) (string, error) {
	mutation := `mutation (
  $update: UpdateCreateInput!
) {
  createUpdate(update: $update) {
    id
  }
}`
	payload := payload{
		Query: mutation,
		Variables: map[string]any{
			"update": map[string]any{
				"title": title,
				"html":  html,
				"account": map[string]any{
					"slug": slug,
				},
			},
		},
	}

	resp, err := doMutation(ctx, payload, token)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	//nolint:tagliatelle
	var envelope struct {
		Data struct {
			CreateUpdate struct {
				ID string `json:"id"`
			} `json:"createUpdate"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("could not decode JSON response: %w", err)
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
	payload := payload{
		Query: mutation,
		Variables: map[string]any{
			"id":       id,
			"audience": "ALL",
		},
	}

	resp, err := doMutation(ctx, payload, token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return err
}

func doMutation(ctx *context.Context, payload payload, token string) (*http.Response, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(p))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Personal-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request to opencollective: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("incorrect response from opencollective: %s", resp.Status)
	}

	return resp, nil
}
