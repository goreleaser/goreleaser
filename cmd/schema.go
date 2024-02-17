package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goreleaser/goreleaser/pkg/config"
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
		Short:             "outputs goreleaser's JSON schema",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(_ *cobra.Command, _ []string) error {
			schema := jsonschema.Reflect(&config.Project{})
			schema.Definitions["FileInfo"] = jsonschema.Reflect(&config.FileInfo{})
			schema.Description = "goreleaser configuration definition file"
			bts, err := json.MarshalIndent(schema, "	", "	")
			if err != nil {
				return fmt.Errorf("failed to create jsonschema: %w", err)
			}
			if root.output == "-" {
				fmt.Println(string(bts))
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(root.output), 0o755); err != nil {
				return fmt.Errorf("failed to write jsonschema file: %w", err)
			}
			if err := os.WriteFile(root.output, bts, 0o666); err != nil {
				return fmt.Errorf("failed to write jsonschema file: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.output, "output", "o", "-", "Where to save the JSONSchema file")
	_ = cmd.MarkFlagFilename("output", "json")

	root.cmd = cmd
	return root
}
