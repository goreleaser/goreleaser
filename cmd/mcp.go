package cmd

import (
	stdctx "context"

	goversion "github.com/caarlos0/go-version"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

type mcpCmd struct {
	cmd *cobra.Command
}

func newMcpCmd(version goversion.Info) *mcpCmd {
	root := &mcpCmd{}
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP (Model Context Protocol) server",
		Long: `Start an MCP server that provides access to GoReleaser functionality.

The MCP server allows AI models and other clients to interact with GoReleaser
through the Model Context Protocol, enabling automated release workflows
and configuration management.`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(_ *cobra.Command, _ []string) error {
			s := server.NewMCPServer("goreleaser", version.GitVersion)
			tool := mcp.NewTool("check_config",
				mcp.WithDescription("Checks a GoReleaser configuration for errors"),
				mcp.WithString("configuration",
					mcp.Required(),
					mcp.Title("GoReleaser config file"),
					mcp.Description("Path the the goreleaser YAML configuration file"),
				),
			)
			s.AddTool(tool, mcpCheck)
			return server.ServeStdio(s)
		},
	}

	root.cmd = cmd
	return root
}

func mcpCheck(ctx stdctx.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input, err := request.RequireString("configuration")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if _, err := config.Load(input); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Configuration is valid!"), nil
}
