package cmd

import (
	stdctx "context"
	"fmt"
	"os"
	"os/exec"

	goversion "github.com/caarlos0/go-version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

type mcpCmd struct {
	cmd *cobra.Command
	bin string
}

func newMcpCmd(version goversion.Info) *mcpCmd {
	root := &mcpCmd{}
	cmd := &cobra.Command{
		Use:               "mcp",
		Short:             "Start a MCP server that provides GoReleaser tools",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(*cobra.Command, []string) error {
			bin, err := os.Executable()
			if err != nil {
				return err
			}
			root.bin = bin
			s := server.NewMCPServer("goreleaser", version.GitVersion)

			s.AddTool(
				mcp.NewTool(
					"check",
					mcp.WithDescription("Checks a GoReleaser configuration for errors"),
					mcp.WithString("configuration",
						mcp.Title("GoReleaser config file"),
						mcp.Description("Path to the goreleaser YAML configuration file. If empty will use the default."),
					),
					mcp.WithReadOnlyHintAnnotation(true),
				),
				root.check,
			)

			s.AddTool(
				mcp.NewTool(
					"healthcheck",
					mcp.WithDescription("Checks if GoReleaser has all the dependencies installed"),
				),
				root.healthcheck,
			)

			s.AddTool(
				mcp.NewTool(
					"build",
					mcp.WithDescription("Builds the current project for the current platform"),
					mcp.WithDestructiveHintAnnotation(true),
				),
				root.build,
			)

			s.AddTool(
				mcp.NewTool(
					"init",
					mcp.WithDescription("Initializes GoReleaser in the current directory"),
					mcp.WithDestructiveHintAnnotation(true),
				),
				root.init,
			)

			return server.ServeStdio(s)
		},
	}

	root.cmd = cmd
	return root
}

func (c *mcpCmd) init(ctx stdctx.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "init").CombinedOutput()
	return mcp.NewToolResultText(string(out)), nil
}

func (c *mcpCmd) healthcheck(ctx stdctx.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "healthcheck").CombinedOutput()
	return mcp.NewToolResultText(string(out)), nil
}

func (c *mcpCmd) build(ctx stdctx.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "build", "--snapshot", "--clean", "--single-target", "-o", ".").CombinedOutput()
	return mcp.NewToolResultText(string(out)), nil
}

func (*mcpCmd) check(_ stdctx.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := request.GetString("configuration", "")
	_, path, err := loadConfigCheck(input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Configuration is invalid", err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Configuration at %q is valid!",
		path,
	)), nil
}
