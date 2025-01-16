package shell_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/shell"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		require.NoError(t, shell.Run(
			testctx.New(),
			"",
			strings.Fields(testlib.Echo("oi")),
			[]string{},
			false,
		))
	})

	t.Run("cmd failed", func(t *testing.T) {
		require.Error(t, shell.Run(
			testctx.New(),
			"",
			strings.Fields(testlib.Exit(1)),
			[]string{},
			false,
		))
	})

	t.Run("cmd with output", func(t *testing.T) {
		testlib.SkipIfWindows(t, "what would be a similar behavior in windows?")
		err := shell.Run(
			testctx.New(),
			".",
			[]string{"sh", "-c", `echo something; exit 1`},
			[]string{},
			true,
		)
		require.EqualError(
			t, err,
			`shell: 'sh -c echo something; exit 1': exit status 1: something`,
		)
	})

	t.Run("with env and dir", func(t *testing.T) {
		testlib.SkipIfWindows(t, "what would be a similar behavior in windows?")
		dir := t.TempDir()
		touch, err := exec.LookPath("touch")
		require.NoError(t, err)
		err = shell.Run(
			testctx.New(),
			dir,
			[]string{"sh", "-c", touch + " $FOO"},
			[]string{"FOO=bar"},
			false,
		)
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(dir, "bar"))
	})
}
