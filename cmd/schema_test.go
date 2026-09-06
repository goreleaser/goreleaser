package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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

	schema := map[string]any{}
	require.NoError(t, json.NewDecoder(outFile).Decode(&schema))
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"].(string))
}

func TestSchemaCommandErrors(t *testing.T) {
	binary := buildGoReleaserBinary(t)
	for _, name := range []string{"jsonschema", "schema"} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, err := runSchemaCommand(t, binary, name, "--not-a-flag")
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, "unknown flag: --not-a-flag")

			var exitErr *exec.ExitError
			require.True(t, errors.As(err, &exitErr))
			require.Equal(t, 1, exitErr.ExitCode())
		})
	}
}

func TestSchemaCommandSuccessWritesJSONToStdout(t *testing.T) {
	binary := buildGoReleaserBinary(t)
	for _, name := range []string{"jsonschema", "schema"} {
		t.Run(name, func(t *testing.T) {
			stdout, _, err := runSchemaCommand(t, binary, name)
			require.NoError(t, err)

			schema := map[string]any{}
			require.NoError(t, json.Unmarshal([]byte(stdout), &schema))
			require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"].(string))
		})
	}
}

func buildGoReleaserBinary(tb testing.TB) string {
	tb.Helper()

	binary := filepath.Join(tb.TempDir(), "goreleaser")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binary, "..")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(tb, cmd.Run(), stderr.String())
	return binary
}

func runSchemaCommand(tb testing.TB, binary string, args ...string) (string, string, error) {
	tb.Helper()

	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
