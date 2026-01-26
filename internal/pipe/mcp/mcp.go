package mcp

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/modelcontextprotocol/registry/cmd/publisher/auth"
	proto "github.com/modelcontextprotocol/registry/cmd/publisher/commands"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
)

const (
	gitHubTokenFilePath   = ".mcpregistry_github_token"   // #nosec:G101
	registryTokenFilePath = ".mcpregistry_registry_token" // #nosec:G101
)

// Pipe for MCP.
type Pipe struct {
	registry       string
	authProviderFn func(registryURL, method, token string) (auth.Provider, error)
}

func New() Pipe {
	return Pipe{
		registry:       proto.DefaultRegistryURL,
		authProviderFn: authProvider,
	}
}

func (Pipe) String() string        { return "mcp registry" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.MCP) || (ctx.Config.MCP.Name == "" && ctx.Config.MCP.GitHub.Name == "")
}

func (Pipe) Default(ctx *context.Context) error {
	// Migrate from deprecated mcp.github to top-level mcp if needed
	if ctx.Config.MCP.GitHub.Name != "" && ctx.Config.MCP.Name == "" {
		deprecate.Notice(ctx, "mcp.github")
		ctx.Config.MCP.Name = ctx.Config.MCP.GitHub.Name
		ctx.Config.MCP.Title = ctx.Config.MCP.GitHub.Title
		ctx.Config.MCP.Description = ctx.Config.MCP.GitHub.Description
		ctx.Config.MCP.Homepage = ctx.Config.MCP.GitHub.Homepage
		ctx.Config.MCP.Packages = ctx.Config.MCP.GitHub.Packages
		ctx.Config.MCP.Transports = ctx.Config.MCP.GitHub.Transports
		ctx.Config.MCP.Disable = ctx.Config.MCP.GitHub.Disable
		ctx.Config.MCP.Repository = ctx.Config.MCP.GitHub.Repository
		ctx.Config.MCP.Auth = ctx.Config.MCP.GitHub.Auth
	}

	ctx.Config.MCP.Auth.Type = cmp.Or(ctx.Config.MCP.Auth.Type, proto.MethodNone)
	return nil
}

func (p Pipe) Publish(ctx *context.Context) error {
	warnExperimental()
	mcp := ctx.Config.MCP

	if err := tmpl.New(ctx).ApplyAll(
		&mcp.Name,
		&mcp.Description,
		&mcp.Title,
		&mcp.Homepage,
		&mcp.Repository.URL,
		&mcp.Repository.Source,
		&mcp.Repository.ID,
		&mcp.Repository.Subfolder,
		&mcp.Auth.Type,
		&mcp.Auth.Token,
	); err != nil {
		return fmt.Errorf("could not apply templates: %w", err)
	}

	provider, err := p.authProviderFn(
		p.registry,
		mcp.Auth.Type,
		mcp.Auth.Token,
	)
	if err != nil {
		return fmt.Errorf("could not login: %w", err)
	}
	if err := provider.Login(ctx); err != nil {
		return fmt.Errorf("could not login: %w", err)
	}
	defer func() {
		// logout...
		_ = os.Remove(gitHubTokenFilePath)
		_ = os.Remove(registryTokenFilePath)
	}()
	token, err := provider.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("could not get token: %w", err)
	}

	var repo *model.Repository
	if mcp.Repository.URL != "" {
		repo = &model.Repository{
			URL:       mcp.Repository.URL,
			Source:    mcp.Repository.Source,
			ID:        mcp.Repository.ID,
			Subfolder: mcp.Repository.Subfolder,
		}
	}
	server := apiv0.ServerJSON{
		Schema:      model.CurrentSchemaURL,
		Name:        mcp.Name,
		Description: mcp.Description,
		Title:       mcp.Title,
		Repository:  repo,
		Version:     ctx.Version,
		WebsiteURL:  mcp.Homepage,
	}
	for _, pkg := range mcp.Packages {
		if err := tmpl.New(ctx).ApplyAll(
			&pkg.Identifier,
		); err != nil {
			return fmt.Errorf("could not apply templates: %w", err)
		}
		version := ctx.Version
		if pkg.RegistryType == "oci" {
			version = ""
		}
		server.Packages = append(server.Packages, model.Package{
			RegistryType: pkg.RegistryType,
			Identifier:   pkg.Identifier,
			Version:      version,
			Transport: model.Transport{
				Type: pkg.Transport.Type,
			},
		})
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("could not serialize request: %w", err)
	}

	publishURL := p.registry + "/v0/publish"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status code %d: %s", resp.StatusCode, string(body))
	}

	var serverResponse apiv0.ServerResponse
	if err := json.Unmarshal(body, &serverResponse); err != nil {
		return fmt.Errorf("could not parse response: %w", err)
	}

	log.
		WithField("name", server.Name).
		WithField("status", serverResponse.Meta.Official.Status).
		Info("published to MCP registry")

	return nil
}

func authProvider(registryURL, method, token string) (auth.Provider, error) {
	switch method {
	case proto.MethodGitHub:
		return auth.NewGitHubATProvider(true, registryURL, token), nil
	case proto.MethodGitHubOIDC:
		return auth.NewGitHubOIDCProvider(registryURL), nil
	case proto.MethodNone:
		return auth.NewNoneProvider(registryURL), nil
	default:
		return nil, fmt.Errorf("unknown auth method: %s", method)
	}
}

func warnExperimental() {
	log.WithField("details", `Keep an eye on the release notes if you wish to rely on this for production builds.
Please provide any feedback you might have at https://github.com/orgs/goreleaser/discussions/6251`).
		Warn(logext.Warning("mcp is experimental and subject to change"))
}
