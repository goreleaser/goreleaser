package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/metadata"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
)

type schemaCmd struct {
	cmd    *cobra.Command
	output string
}

func newSchemaCmd() *schemaCmd {
	root := &schemaCmd{}
	cmd := &cobra.Command{
		Use:               "jsonschema",
		Aliases:           []string{"schema"},
		Short:             "Outputs goreleaser's configuration JSON schema",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(*cobra.Command, []string) error {
			schema := jsonschema.Reflect(&config.Project{})
			schema.Definitions["FileInfo"] = jsonschema.Reflect(&config.FileInfo{})
			schema.Description = "goreleaser configuration definition file"
			return outputSchema(schema, root.output)
		},
	}

	cmd.PersistentFlags().StringVarP(&root.output, "output", "o", "-", "Where to save the JSONSchema file")
	_ = cmd.MarkPersistentFlagFilename("output", "json")

	cmd.AddCommand(newArtifactsSchemaCmd(root))
	cmd.AddCommand(newMetadataSchemaCmd(root))

	root.cmd = cmd
	return root
}

func newArtifactsSchemaCmd(root *schemaCmd) *cobra.Command {
	return &cobra.Command{
		Use:               "artifacts",
		Short:             "Outputs goreleaser's build artifacts JSON schema",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(*cobra.Command, []string) error {
			schema := jsonschema.Reflect([]artifact.Artifact{})
			schema.Description = "goreleaser build artifacts definition file"
			return outputSchema(schema, root.output)
		},
	}
}

func newMetadataSchemaCmd(root *schemaCmd) *cobra.Command {
	return &cobra.Command{
		Use:               "metadata",
		Short:             "Outputs goreleaser's build metadata JSON schema",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(*cobra.Command, []string) error {
			schema := jsonschema.Reflect(&metadata.Metadata{})
			schema.Description = "goreleaser build metadata definition file"
			return outputSchema(schema, root.output)
		},
	}
}

func outputSchema(schema any, output string) error {
	bts, err := json.MarshalIndent(schema, "	", "	")
	if err != nil {
		return fmt.Errorf("failed to create jsonschema: %w", err)
	}
	if output == "-" {
		fmt.Println(string(bts))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("failed to write jsonschema file: %w", err)
	}
	if err := os.WriteFile(output, bts, 0o666); err != nil {
		return fmt.Errorf("failed to write jsonschema file: %w", err)
	}
	return nil

}
