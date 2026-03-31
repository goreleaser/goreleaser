// Package opencollective announces the release to Open Collective.
package opencollective

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultTitleTemplate   = `{{ .Tag }}`
	defaultMessageTemplate = `{{ .ProjectName }} {{ .Tag }} is out!<br/>Check it out at <a href="{{ .ReleaseURL }}">{{ .ReleaseURL }}</a>`
	defaultEndpoint        = "https://api.opencollective.com/graphql/v2"
)

type Pipe struct{}

func (Pipe) String() string { return "opencollective" }

func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.OpenCollective.Enabled)
	return !enable || ctx.Config.Announce.OpenCollective.Slug == "", err
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

func (p Pipe) Announce(ctx *context.Context) error {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Announce.OpenCollective.TitleTemplate)
	if err != nil {
		return err
	}
	html, err := tmpl.New(ctx).Apply(ctx.Config.Announce.OpenCollective.MessageTemplate)
	if err != nil {
		return err
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return err
	}

	log.Infof("posting: %q | %q", title, html)

	c := client{
		endpoint: defaultEndpoint,
		token:    cfg.Token,
	}

	id, err := c.createUpdate(ctx, title, html, ctx.Config.Announce.OpenCollective.Slug)
	if err != nil {
		return err
	}

	return c.publishUpdate(ctx, id)
}

type client struct {
	endpoint string
	token    string
}

type payload struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// graphqlResponse decodes GraphQL responses and checks for errors.
// GraphQL APIs return HTTP 200 even on mutation failures — errors are in the
// response body, not the status code.
type graphqlResponse struct {
	Errors []graphqlError `json:"errors"`
}

type graphqlError struct {
	Message string `json:"message"`
}

func (r graphqlResponse) err() error {
	if len(r.Errors) == 0 {
		return nil
	}
	msgs := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		msgs = append(msgs, e.Message)
	}
	return fmt.Errorf("opencollective graphql error: %s", strings.Join(msgs, "; "))
}

func (c client) createUpdate(ctx *context.Context, title, html, slug string) (string, error) {
	mutation := `mutation (
  $update: UpdateCreateInput!
) {
  createUpdate(update: $update) {
    id
  }
}`
	p := payload{
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

	body, err := c.doMutation(ctx, p)
	if err != nil {
		return "", err
	}

	//nolint:tagliatelle
	var envelope struct {
		graphqlResponse
		Data struct {
			CreateUpdate struct {
				ID string `json:"id"`
			} `json:"createUpdate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("could not decode JSON response: %w", err)
	}
	if err := envelope.err(); err != nil {
		return "", err
	}
	if envelope.Data.CreateUpdate.ID == "" {
		return "", errors.New("opencollective returned empty update id")
	}

	return envelope.Data.CreateUpdate.ID, nil
}

func (c client) publishUpdate(ctx *context.Context, id string) error {
	mutation := `mutation (
  $id: String!
  $audience: UpdateAudience
) {
  publishUpdate(id: $id, notificationAudience: $audience) {
    id
  }
}`
	p := payload{
		Query: mutation,
		Variables: map[string]any{
			"id":       id,
			"audience": "ALL",
		},
	}

	body, err := c.doMutation(ctx, p)
	if err != nil {
		return err
	}

	var envelope graphqlResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("could not decode JSON response: %w", err)
	}
	return envelope.err()
}

func (c client) doMutation(ctx *context.Context, payload payload) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(p))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Personal-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request to opencollective: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response from opencollective: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incorrect response from opencollective: %s — %s", resp.Status, string(body))
	}

	return body, nil
}
