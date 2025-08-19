package cmd

import (
	stdctx "context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	goversion "github.com/caarlos0/go-version"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
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
					mcp.WithDescription("Checks a GoReleaser configuration for errors or deprecations"),
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

var instructions = map[string]string{
	"archives.builds":                  "replace `builds` with `ids`",
	"archives.format":                  "replace `format` with `formats` and make its value an array",
	"archives.format_overrides.format": "replace `format` with `formats` and make its value an array",
	"builds.gobinary":                  "rename `gobinary` to `tool`",
	"homebrew_casks.manpage":           "replace `manpage` with `manpages`, and make its value an array",
	"kos.repository":                   "replace `repository` with `repositories`, and make its value an array",
	"kos.sbom":                         "the value of `sbom` can only be `spdx` or `none`, set it to `spdx` if there's any other value there",
	"nfpms.builds":                     "rename `builds` to `ids`",
	"nightly.name_template":            "rename `name_template` to `version_template`",
	"snaps.builds":                     "rename `builds` to `ids`",
	"snapshot.name_template":           "rename `name_template` to `version_template`",
}

func (*mcpCmd) check(ctx stdctx.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := request.GetString("configuration", "")
	cfg, path, err := loadConfigCheck(input)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("Configuration is invalid", err), nil
	}

	gctx := context.Wrap(ctx, cfg)
	if err := (defaults.Pipe{}).Run(gctx); err != nil {
		return mcp.NewToolResultErrorFromErr("Configuration is invalid", err), nil
	}

	if gctx.Deprecated {
		var sb strings.Builder
		sb.WriteString("Configuration is valid, but uses the following deprecated properties:\n")
		for _, key := range slices.Collect(maps.Keys(gctx.NotifiedDeprecations)) {
			sb.WriteString(fmt.Sprintf("## %s\n\nInstructions: %s\n\n", key, instructions[key]))
		}
		return mcp.NewToolResultText(sb.String()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Configuration at %q is valid!",
		path,
	)), nil
}
