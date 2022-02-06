package shell_test

import (
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		require.NoError(t, shell.Run(context.New(config.Project{}), "", []string{"echo", "oi"}, []string{}, false))
	})

	t.Run("cmd failed", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(context.New(config.Project{}), "", []string{"sh", "-c", "exit 1"}, []string{}, false),
			`failed to run 'sh -c exit 1': exit status 1`,
		)
	})

	t.Run("cmd with output", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(context.New(config.Project{}), "", []string{"sh", "-c", `echo something; exit 1`}, []string{}, true),
			`failed to run 'sh -c echo something; exit 1': exit status 1`,
		)
	})

	t.Run("with env and dir", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, shell.Run(context.New(config.Project{}), dir, []string{"sh", "-c", "touch $FOO"}, []string{"FOO=bar"}, false))
		require.FileExists(t, filepath.Join(dir, "bar"))
	})
}
