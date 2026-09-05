package config

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func TestSlackJSONSchema(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(jsonschema.Reflect(&Project{}))
	require.NoError(t, err)
	schema, err := validator.CompileString("schema.json", string(data))
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		config io.Reader
	}{
		{
			name: "blocks and attachments",
			config: strings.NewReader(`
announce:
  slack:
    blocks:
      - type: divider
    attachments:
      - text: hello
`),
		},
		{name: "advanced blocks", config: goodBlocksSlackConf()},
		{name: "advanced attachments", config: goodAttachmentsSlackConf()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			project, err := LoadReader(tc.config)
			require.NoError(t, err)
			data, err := json.Marshal(map[string]any{
				"announce": map[string]any{"slack": project.Announce.Slack},
			})
			require.NoError(t, err)
			var document any
			require.NoError(t, json.Unmarshal(data, &document))
			require.NoError(t, schema.Validate(document))
		})
	}

	for _, field := range []string{"blocks", "attachments"} {
		t.Run("invalid "+field, func(t *testing.T) {
			t.Parallel()

			require.Error(t, schema.Validate(map[string]any{
				"announce": map[string]any{
					"slack": map[string]any{field: []any{"not an object"}},
				},
			}))
		})
	}
}

func TestSignJSONSchema(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(jsonschema.Reflect(&Project{}))
	require.NoError(t, err)
	schema, err := validator.CompileString("schema.json", string(data))
	require.NoError(t, err)

	for _, field := range []string{"signs", "docker_signs"} {
		for _, selector := range []string{"none", "checksum", "invalid"} {
			t.Run(field+"/"+selector, func(t *testing.T) {
				t.Parallel()

				err := schema.Validate(map[string]any{
					field: []any{map[string]any{"artifacts": selector}},
				})
				if selector == "invalid" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}
