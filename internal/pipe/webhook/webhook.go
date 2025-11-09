// Package webhook announces the new release by sending a webhook.
package webhook

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	defaultMessageTemplate = `{ "message": "{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}"}`
	ContentTypeHeaderKey   = "Content-Type"
	UserAgentHeaderKey     = "User-Agent"
	UserAgentHeaderValue   = "goreleaser"
	AuthorizationHeaderKey = "Authorization"
	DefaultContentType     = "application/json; charset=utf-8"
)

var defaultExpectedStatusCodes = []int{
	http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent,
}

type Pipe struct{}

func (Pipe) String() string { return "webhook" }
func (Pipe) Skip(ctx *context.Context) (bool, error) {
	enable, err := tmpl.New(ctx).Bool(ctx.Config.Announce.Webhook.Enabled)
	return !enable, err
}

type Config struct {
	BasicAuthHeader   string `env:"BASIC_AUTH_HEADER_VALUE"`
	BearerTokenHeader string `env:"BEARER_TOKEN_HEADER_VALUE"`
}

func (p Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Announce.Webhook.MessageTemplate == "" {
		ctx.Config.Announce.Webhook.MessageTemplate = defaultMessageTemplate
	}
	if ctx.Config.Announce.Webhook.ContentType == "" {
		ctx.Config.Announce.Webhook.ContentType = DefaultContentType
	}
	if len(ctx.Config.Announce.Webhook.ExpectedStatusCodes) == 0 {
		ctx.Config.Announce.Webhook.ExpectedStatusCodes = defaultExpectedStatusCodes
	}
	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	endpointURLConfig, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Webhook.EndpointURL)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	if len(endpointURLConfig) == 0 {
		return fmt.Errorf("%s: no endpoint url", p)
	}

	if _, err := url.ParseRequestURI(endpointURLConfig); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	endpointURL, err := url.Parse(endpointURLConfig)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Webhook.MessageTemplate)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}

	log.Infof("posting: '%s'", msg)
	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.Announce.Webhook.SkipTLSVerify,
	}

	client := &http.Client{
		Transport: customTransport,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), strings.NewReader(msg))
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	req.Header.Add(ContentTypeHeaderKey, ctx.Config.Announce.Webhook.ContentType)
	req.Header.Add(UserAgentHeaderKey, UserAgentHeaderValue)

	if cfg.BasicAuthHeader != "" {
		log.Debugf("set basic auth header")
		req.Header.Add(AuthorizationHeaderKey, cfg.BasicAuthHeader)
	} else if cfg.BearerTokenHeader != "" {
		log.Debugf("set bearer token header")
		req.Header.Add(AuthorizationHeaderKey, cfg.BearerTokenHeader)
	}

	for key, value := range ctx.Config.Announce.Webhook.Headers {
		log.Debugf("Header Key %s / Value %s", key, value)
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	defer resp.Body.Close()

	if !slices.Contains(ctx.Config.Announce.Webhook.ExpectedStatusCodes, resp.StatusCode) {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("request failed with status %v", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	log.Infof("Post OK: '%v'", resp.StatusCode)
	log.Infof("Response : %v\n", string(body))
	return nil
}
