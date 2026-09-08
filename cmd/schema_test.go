package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/caarlos0/log"
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

	schema := map[string]any{}
	require.NoError(t, json.NewDecoder(outFile).Decode(&schema))
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"].(string))
}

func TestSchemaCommandErrors(t *testing.T) {
	for _, name := range []string{"jsonschema", "schema"} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			previousLog := log.Log
			log.Log = log.New(&stderr)
			log.SetLevel(log.InfoLevel)
			t.Cleanup(func() {
				log.Log = previousLog
			})

			mem := &exitMemento{}
			cmd := newRootCmd(testversion, mem.Exit)
			cmd.Execute([]string{name, "--not-a-flag"})

			require.Equal(t, 1, mem.code)
			require.Contains(t, stderr.String(), "unknown flag: --not-a-flag")
		})
	}
}

func TestSchemaCommandSuccessWritesJSONToStdout(t *testing.T) {
	for _, name := range []string{"jsonschema", "schema"} {
		t.Run(name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "schema-stdout")
			require.NoError(t, err)

			stdout := os.Stdout
			os.Stdout = file
			t.Cleanup(func() {
				os.Stdout = stdout
			})

			mem := &exitMemento{}
			cmd := newRootCmd(testversion, mem.Exit)
			cmd.Execute([]string{name})

			require.NoError(t, file.Close())
			out, err := os.ReadFile(file.Name())
			require.NoError(t, err)
			require.Equal(t, 0, mem.code)

			schema := map[string]any{}
			require.NoError(t, json.Unmarshal(out, &schema))
			require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"].(string))
		})
	}
}
