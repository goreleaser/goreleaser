package shell_test

import (
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		require.NoError(t, shell.Run(testctx.New(), "", []string{"echo", "oi"}, []string{}, false))
	})

	t.Run("cmd failed", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(testctx.New(), "", []string{"sh", "-c", "exit 1"}, []string{}, false),
			`failed to run 'sh -c exit 1': exit status 1`,
		)
	})

	t.Run("cmd with output", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(testctx.New(), "", []string{"sh", "-c", `echo something; exit 1`}, []string{}, true),
			`failed to run 'sh -c echo something; exit 1': exit status 1`,
		)
	})

	t.Run("with env and dir", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, shell.Run(testctx.New(), dir, []string{"sh", "-c", "touch $FOO"}, []string{"FOO=bar"}, false))
		require.FileExists(t, filepath.Join(dir, "bar"))
	})
}
