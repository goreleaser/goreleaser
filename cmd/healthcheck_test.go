package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthcheckSystem(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestHealthcheckConfigThatDoesNotExist(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/nope.yml"})
	require.ErrorIs(t, cmd.cmd.Execute(), os.ErrNotExist)
}

func TestHealthcheckMissingTool(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/missing_tool.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "one or more checks failed")
}

func TestHealthcheckBlankSignerIsSkipped(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/blank_tool.yml"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestHealthcheckQuier(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/good.yml", "--quiet"})
	require.NoError(t, cmd.cmd.Execute())
}

func TestCheckPath(t *testing.T) {
	checked := map[string]bool{}
	require.NoError(t, checkPath(t.Context(), checked, "go"))
	require.NoError(t, checkPath(t.Context(), checked, "git version"))
	// `go` rather than `docker`: this case is about a tool that is on PATH but
	// whose command fails, and docker is not guaranteed to be installed -- when
	// it is not, the LookPath branch answers instead and the case proves
	// nothing. It is also slow to refuse: 5.26s on the windows job.
	require.Error(t, checkPath(t.Context(), checked, "go something-invalid"))
	require.Error(t, checkPath(t.Context(), checked, "some invalid command"))
	require.NoError(t, checkPath(t.Context(), checked, " \t "))
	require.Error(t, checkPath(t.Context(), checked, `"unterminated`))
	// shell syntax alone parses to no arguments at all.
	require.ErrorIs(t, checkPath(t.Context(), checked, "|"), exec.ErrNotFound)
}

func TestCheckPathChecksEachToolOnce(t *testing.T) {
	checked := map[string]bool{}
	require.Error(t, checkPath(t.Context(), checked, "some invalid command"))
	// second call is deduped by the cache, so it reports no error even though
	// the tool is still missing.
	require.NoError(t, checkPath(t.Context(), checked, "some invalid command"))
	// a cache of its own sees the failure again.
	require.Error(t, checkPath(t.Context(), map[string]bool{}, "some invalid command"))
}

func TestCheckPathLiteralExecutable(t *testing.T) {
	for _, name := range []string{"signer's-tool", "signer's tool"} {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		path := filepath.Join(t.TempDir(), name)
		require.NoError(t, os.WriteFile(path, nil, 0o755))
		for _, tool := range []string{path, fmt.Sprintf("%q", path)} {
			t.Run(tool, func(t *testing.T) {
				require.NoError(t, checkPath(t.Context(), map[string]bool{}, tool))
			})
		}
	}
}
