package nodesea

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunNPMBuild_BadPackageJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "package.json"),
		[]byte(`{not json`), 0o644))
	err := RunNPMBuild(t.Context(), dir, nil, nil, nil)
	require.Error(t, err)
}

// TestRunNPMBuild exercises the auto-bundle entrypoint against a fake
// `npm` shipped on PATH — a tiny shell script that records its args
// into a file so the test can assert on what it received.
func TestRunNPMBuild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh fake npm")
	}

	t.Run("runs npm run build when scripts.build is declared", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "package.json"),
			[]byte(`{"scripts":{"build":"esbuild ..."}}`), 0o644))
		fakeNPMOnPath(t, dir, 0)

		var stdout, stderr bytes.Buffer
		require.NoError(t, RunNPMBuild(t.Context(), dir, nil, &stdout, &stderr))

		got, err := os.ReadFile(filepath.Join(dir, "calls.log"))
		require.NoError(t, err)
		require.Equal(t, "run build\n", string(got))
	})

	t.Run("silent skip when no package.json", func(t *testing.T) {
		require.NoError(t, RunNPMBuild(t.Context(), t.TempDir(), nil, nil, nil))
	})

	t.Run("silent skip when scripts.build missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "package.json"),
			[]byte(`{"scripts":{"test":"vitest"}}`), 0o644))
		// no fake npm on PATH; if RunNPMBuild attempted to spawn npm
		// the test would either fail loudly or shell out to the host
		// npm — both undesirable. Empty env keeps PATH unset so any
		// spawn attempt fails fast.
		require.NoError(t, RunNPMBuild(t.Context(), dir, []string{}, nil, nil))
	})

	t.Run("propagates non-zero exit", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "package.json"),
			[]byte(`{"scripts":{"build":"x"}}`), 0o644))
		fakeNPMOnPath(t, dir, 1)

		err := RunNPMBuild(t.Context(), dir, nil, nil, nil)
		require.Error(t, err)
	})
}

// fakeNPMOnPath drops a fake `npm` shell script into a fresh temp
// dir, prepends it to PATH for the duration of the test, and writes
// an entry to logDir/calls.log on each invocation.
func fakeNPMOnPath(t *testing.T, logDir string, exitCode int) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "npm")
	script := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + logDir + "/calls.log\"\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
