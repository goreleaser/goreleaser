package webhook

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	defaultMessageTemplate = `{ "message": "{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}"}`
	ContentTypeHeaderKey   = "Content-Type"
	UserAgentHeaderKey     = "User-Agent"
	UserAgentHeaderValue   = "gorleaser"
	AuthorizationHeaderKey = "Authorization"
	DefaultContentType     = "application/json; charset=utf-8"
)

type Pipe struct{}

func (Pipe) String() string                 { return "webhook" }
func (Pipe) Skip(ctx *context.Context) bool { return !ctx.Config.Announce.Webhook.Enabled }

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
	return nil
}

func (p Pipe) Announce(ctx *context.Context) error {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
	}

	endpointURLConfig, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Webhook.EndpointURL)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
	}
	if len(endpointURLConfig) == 0 {
		return errors.New("announce: failed to announce to webhook: no endpoint url")
	}

	if _, err := url.ParseRequestURI(endpointURLConfig); err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
	}
	endpointURL, err := url.Parse(endpointURLConfig)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
	}

	msg, err := tmpl.New(ctx).Apply(ctx.Config.Announce.Webhook.MessageTemplate)
	if err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %s", err)
	}

	log.Infof("posting: '%s'", msg)
	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.Announce.Webhook.SkipTLSVerify,
	}

	client := &http.Client{
		Transport: customTransport,
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL.String(), strings.NewReader(msg))
	if err != nil {
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
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
		return fmt.Errorf("announce: failed to announce to webhook: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		log.Infof("Post OK: '%v'", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		log.Infof("Response : %v\n", string(body))
		return nil
	default:
		return fmt.Errorf("request failed with status %v", resp.Status)
	}
}
