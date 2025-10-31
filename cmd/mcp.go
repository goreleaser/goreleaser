package cmd

import (
	stdctx "context"
	"fmt"
	"maps"
	"slices"
	"strings"

	goversion "github.com/caarlos0/go-version"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/goreleaser/v2/www/docs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

const updatePrompt = `Let's update the goreleaser configuration to latest.

We can use the goreleaser check command to grab the deprecation notices and how to fix them.

If that's not enough, use the documentation resources to find out more details.
The resource paths to look at are:

- docs://deprecations.md
- docs://customization/{feature name}.md
- docs://old-deprecations.md (this one only if updating between goreleaser major versions)
`

type mcpCmd struct {
	cmd *cobra.Command
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
			server := mcp.NewServer(&mcp.Implementation{
				Name:    "goreleaser",
				Version: version.GitVersion,
			}, nil)

			server.AddPrompt(&mcp.Prompt{
				Name:  "update_config",
				Title: "Update GoReleaser Configuration",
			}, func(ctx stdctx.Context, gpr *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				return &mcp.GetPromptResult{
					Messages: []*mcp.PromptMessage{
						{Content: &mcp.TextContent{Text: updatePrompt}},
					},
				}, nil
			})

			server.AddResourceTemplate(&mcp.ResourceTemplate{
				URITemplate: "docs://{f}",
			}, func(ctx stdctx.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				uri := strings.TrimPrefix(req.Params.URI, "docs://")
				bts, err := docs.FS.ReadFile(uri)
				if err != nil {
					return nil, mcp.ResourceNotFoundError(uri)
				}
				return &mcp.ReadResourceResult{
					Contents: []*mcp.ResourceContents{{
						URI:  req.Params.URI,
						Text: string(bts),
					}},
				}, nil
			})

			mcp.AddTool(server, &mcp.Tool{
				Name:        "check",
				Description: "Checks a GoReleaser configuration for errors or deprecations",
			}, root.check)

			return server.Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}

	root.cmd = cmd
	return root
}

var instructions = map[string]string{
	"archives.builds":                  "replace `builds` with `ids`",
	"archives.format":                  "replace `format` with `formats` and make its value an array",
	"archives.format_overrides.format": "replace `format` with `formats` and make its value an array",
	"builds.gobinary":                  "rename `gobinary` to `tool`",
	"homebrew_casks.manpage":           "replace `manpage` with `manpages`, and make its value an array",
	"homebrew_casks.binary":            "replace `binary` with `binaries`, and make its value an array",
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
