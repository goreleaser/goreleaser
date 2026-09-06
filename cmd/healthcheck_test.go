package cmd

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func TestHealthcheckInvalidBlankTool(t *testing.T) {
	cmd := newHealthcheckCmd()
	cmd.cmd.SetArgs([]string{"-f", "testdata/blank_tool.yml"})
	require.EqualError(t, cmd.cmd.Execute(), "one or more checks failed")
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
	require.ErrorIs(t, checkPath(t.Context(), checked, " \t "), exec.ErrNotFound)
}

func TestCheckPathWithSpacesInToolPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tools with spaces")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	tool := filepath.Join(dir, filepath.Base(os.Args[0]))
	src, err := os.Open(os.Args[0])
	require.NoError(t, err)
	defer src.Close()
	dst, err := os.OpenFile(tool, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	require.NoError(t, err)
	_, err = io.Copy(dst, src)
	require.NoError(t, err)
	require.NoError(t, dst.Close())

	require.NoError(t, checkPath(t.Context(), map[string]bool{}, tool))
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
