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
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			bin, err := os.Executable()
			if err != nil {
				return err
			}
			root.bin = bin

			server := mcp.NewServer(&mcp.Implementation{
				Name:    "goreleaser",
				Version: version.GitVersion,
			}, nil)

			mcp.AddTool(server, &mcp.Tool{
				Name:        "check",
				Description: "Checks a GoReleaser configuration for errors or deprecations",
			}, root.check)

			mcp.AddTool(server, &mcp.Tool{
				Name:        "healthcheck",
				Description: "Checks if GoReleaser has all the dependencies installed",
			}, root.healthcheck)

			mcp.AddTool(server, &mcp.Tool{
				Name:        "build",
				Description: "Builds the current project for the current platform",
			}, root.build)

			mcp.AddTool(server, &mcp.Tool{
				Name:        "init",
				Description: "Initializes GoReleaser in the current directory",
			}, root.init)

			return server.Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}

	root.cmd = cmd
	return root
}

type (
	initArgs   struct{}
	initOutput struct {
		Output string `json:"output"`
	}
)

func (c *mcpCmd) init(ctx stdctx.Context, _ *mcp.CallToolRequest, _ initArgs) (*mcp.CallToolResult, initOutput, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "init").CombinedOutput()
	return nil, initOutput{Output: string(out)}, nil
}

type (
	healthcheckArgs   struct{}
	healthcheckOutput struct {
		Output string `json:"output"`
	}
)

func (c *mcpCmd) healthcheck(ctx stdctx.Context, _ *mcp.CallToolRequest, _ healthcheckArgs) (*mcp.CallToolResult, healthcheckOutput, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "healthcheck").CombinedOutput()
	return nil, healthcheckOutput{Output: string(out)}, nil
}

type (
	buildArgs   struct{}
	buildOutput struct {
		Output string `json:"output"`
	}
)

func (c *mcpCmd) build(ctx stdctx.Context, _ *mcp.CallToolRequest, _ buildArgs) (*mcp.CallToolResult, buildOutput, error) {
	out, _ := exec.CommandContext(ctx, c.bin, "build", "--snapshot", "--clean", "--single-target", "-o", ".").CombinedOutput()
	return nil, buildOutput{Output: string(out)}, nil
}

var instructions = map[string]string{
	"archives.builds":                  "replace `builds` with `ids`",
	"archives.format":                  "replace `format` with `formats` and make its value an array",
	"archives.format_overrides.format": "replace `format` with `formats` and make its value an array",
	"builds.gobinary":                  "rename `gobinary` to `tool`",
	"homebrew_casks.manpage":           "replace `manpage` with `manpages`, and make its value an array",
	"homebrew_casks.conflicts.formula": "remove the `formula: <name>` from the `conflicts` list",
	"kos.repository":                   "replace `repository` with `repositories`, and make its value an array",
	"kos.sbom":                         "the value of `sbom` can only be `spdx` or `none`, set it to `spdx` if there's any other value there",
	"nfpms.builds":                     "rename `builds` to `ids`",
	"nightly.name_template":            "rename `name_template` to `version_template`",
	"snaps.builds":                     "rename `builds` to `ids`",
	"snapshot.name_template":           "rename `name_template` to `version_template`",
}

type (
	checkArgs struct {
		Configuration string `json:"configuration" jsonschema:"Path to the goreleaser YAML configuration file. If empty will use the default."`
	}
	checkOutput struct {
		Message string `json:"message"`
	}
)

func (*mcpCmd) check(ctx stdctx.Context, _ *mcp.CallToolRequest, args checkArgs) (*mcp.CallToolResult, checkOutput, error) {
	cfg, path, err := loadConfigCheck(args.Configuration)
	if err != nil {
		return nil, checkOutput{}, fmt.Errorf("configuration is invalid: %w", err)
	}

	gctx := context.Wrap(ctx, cfg)
	if err := (defaults.Pipe{}).Run(gctx); err != nil {
		return nil, checkOutput{}, fmt.Errorf("configuration is invalid: %w", err)
	}

	if gctx.Deprecated {
		var sb strings.Builder
		sb.WriteString("Configuration is valid, but uses the following deprecated properties:\n")
		for _, key := range slices.Collect(maps.Keys(gctx.NotifiedDeprecations)) {
			sb.WriteString(fmt.Sprintf("## %s\n\nInstructions: %s\n\n", key, instructions[key]))
		}
		return nil, checkOutput{Message: sb.String()}, nil
	}

	return nil, checkOutput{
		Message: fmt.Sprintf("Configuration at %q is valid!", path),
	}, nil
}
