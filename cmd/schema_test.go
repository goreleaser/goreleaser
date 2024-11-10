package cmd

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSchema(t *testing.T) {
	cmd := newSchemaCmd().cmd
	dir := t.TempDir()
	destination := path.Join(dir, "schema.json")
	cmd.SetArgs([]string{"--output", destination})
	require.NoError(t, cmd.Execute())

	outFile, err := os.Open(destination)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, outFile.Close())
	})

	schema := map[string]interface{}{}
	require.NoError(t, json.NewDecoder(outFile).Decode(&schema))
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"].(string))
}
