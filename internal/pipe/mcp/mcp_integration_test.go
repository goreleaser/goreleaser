package mcp

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestPublishIntegration tests publishing to the MCP staging registry.
// This test is skipped by default and requires:
// - RUN_INTEGRATION_TESTS=1 environment variable
// - GITHUB_TOKEN environment variable with a GitHub token
// - Running in GitHub Actions with OIDC configured, OR
// - A GitHub token that has been manually exchanged for a registry JWT
//
// To run: RUN_INTEGRATION_TESTS=1 GITHUB_TOKEN=<token> go test -v -run TestPublishIntegration
func TestPublishIntegration(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "goreleaser-mcp",
		MCP: config.MCP{
			Name:        "io.github.goreleaser/mcp",
			Description: "GoReleaser MCP server for build automation",
			// TODO: Homepage:    "https://goreleaser.com",
			Repository: config.MCPRepository{
				Source: "github",
				URL:    "https://github.com/goreleaser/mcp",
			},
			Packages: []config.MCPPackage{
				{
					RegistryType: "oci",
					Identifier:   "ghcr.io/goreleaser/mcp:{{.Version}}",
					Transport: config.MCPTransport{
						Type: "stdio",
					},
				},
			},
			Auth: config.MCPAuth{
				Type:  "github",
				Token: token,
			},
		},
	})
	ctx.Version = "v0.0.1"

	pipe := Pipe{
		registry:       "https://staging.registry.modelcontextprotocol.io",
		authProviderFn: authProvider,
	}

	require.NoError(t, pipe.Default(ctx))
	require.NoError(t, pipe.Publish(ctx))
}
