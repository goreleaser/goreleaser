package mcp

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/deprecate"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/summary"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/modelcontextprotocol/registry/cmd/publisher/auth"
	proto "github.com/modelcontextprotocol/registry/cmd/publisher/commands"
	apiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
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
		ctx.Config.MCP.MCPDetails = ctx.Config.MCP.GitHub
	}

	ctx.Config.MCP.Auth.Type = cmp.Or(ctx.Config.MCP.Auth.Type, proto.MethodNone)
	return nil
}

func (p Pipe) Publish(ctx *context.Context) error {
	mcp := ctx.Config.MCP

	disable, err := tmpl.New(ctx).Apply(mcp.Disable)
	if err != nil {
		return fmt.Errorf("could not evaluate mcp.disable: %w", err)
	}
	if strings.TrimSpace(disable) == "true" {
		return pipe.Skip("mcp.disable is set")
	}
	if strings.TrimSpace(disable) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' disable, skipping mcp publish")
	}

	warnExperimental()

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
			Subfolder: cleanSubfolder(mcp.Repository.Subfolder),
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
			&pkg.Transport.URL,
		); err != nil {
			return fmt.Errorf("could not apply templates: %w", err)
		}
		fileSHA256, err := mcpbFileSHA256(ctx, pkg)
		if err != nil {
			return err
		}
		version := ctx.Version
		if pkg.RegistryType == "oci" {
			version = ""
		}
		server.Packages = append(server.Packages, model.Package{
			RegistryType: pkg.RegistryType,
			Identifier:   pkg.Identifier,
			Version:      version,
			FileSHA256:   fileSHA256,
			Transport: model.Transport{
				Type: pkg.Transport.Type,
				URL:  pkg.Transport.URL,
			},
		})
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("could not serialize request: %w", err)
	}

	publishURL := p.registry + "/v0/publish"

	return retryx.Do(ctx, ctx.Config.Retry, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, publishURL, bytes.NewReader(jsonData))
		if err != nil {
			return retryx.Unrecoverable(fmt.Errorf("could not create request: %w", err))
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return retryx.HTTP(fmt.Errorf("could not send request: %w", err), resp)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("could not read response: %w", err)
		}

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			return retryx.HTTP(fmt.Errorf("got status code %d: %s", resp.StatusCode, string(body)), resp)
		}

		var serverResponse apiv0.ServerResponse
		if err := json.Unmarshal(body, &serverResponse); err != nil {
			return fmt.Errorf("could not parse response: %w", err)
		}

		log.
			WithField("name", server.Name).
			WithField("status", serverResponse.Meta.Official.Status).
			Info("published to MCP registry")
		summary.Appendf("Published `%s` to the MCP registry", server.Name)

		return nil
	}, retryx.IsRetriable)
}

func mcpbFileSHA256(ctx *context.Context, pkg config.MCPPackage) (string, error) {
	if pkg.RegistryType != "mcpb" {
		return "", nil
	}
	artifactName, err := mcpbArtifactName(pkg.Identifier)
	if err != nil {
		return "", err
	}
	art, err := findArtifact(ctx, artifactName)
	if err != nil {
		return "", fmt.Errorf("mcpb package %q: %w", pkg.Identifier, err)
	}
	checksum, err := art.Checksum("sha256")
	if err != nil {
		return "", fmt.Errorf("mcpb package %q: %w", pkg.Identifier, err)
	}
	return checksum, nil
}

func mcpbArtifactName(identifier string) (string, error) {
	u, err := url.Parse(identifier)
	if err != nil {
		return "", fmt.Errorf("parse mcpb package identifier: %w", err)
	}
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "", fmt.Errorf("mcpb package %q does not identify an artifact file", identifier)
	}
	name, err = url.PathUnescape(name)
	if err != nil {
		return "", fmt.Errorf("parse mcpb package identifier: %w", err)
	}
	return name, nil
}

func findArtifact(ctx *context.Context, name string) (*artifact.Artifact, error) {
	var matches []*artifact.Artifact
	for _, art := range ctx.Artifacts.Filter(artifact.ByTypes(artifact.ReleaseUploadableTypes()...)).List() {
		if art.Name == name {
			matches = append(matches, art)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("could not find artifact %q", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("found multiple artifacts named %q", name)
	}
}

// cleanSubfolder normalizes the repository subfolder path so it passes the MCP
// registry validation, which rejects paths with "./" prefixes, trailing
// slashes, and similar non-clean forms.
func cleanSubfolder(subfolder string) string {
	if cleaned := path.Clean(subfolder); cleaned != "." {
		return cleaned
	}
	return ""
}

func authProvider(registryURL, method, token string) (auth.Provider, error) {
	switch method {
	case proto.MethodGitHub:
		return auth.NewGitHubATProvider(registryURL, token), nil
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
Please provide any feedback you might have at https://github.com/goreleaser/goreleaser/discussions/6251`).
		Warn(logext.Warning("mcp is experimental and subject to change"))
}
