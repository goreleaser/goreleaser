package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/jsonschema"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/muesli/coral"
)

type schemaCmd struct {
	cmd    *coral.Command
	output string
}

func newSchemaCmd() *schemaCmd {
	root := &schemaCmd{}
	cmd := &coral.Command{
		Use:           "jsonschema",
		Aliases:       []string{"schema"},
		Short:         "outputs goreleaser's JSON schema",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          coral.NoArgs,
		RunE: func(cmd *coral.Command, args []string) error {
			schema := jsonschema.Reflect(&config.Project{})
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

	cmd.Flags().StringVarP(&root.output, "output", "o", "-", "where to save the json schema")

	root.cmd = cmd
	return root
}
