//go:build windows
// +build windows

package shell_test

import (
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/shell"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestRunCommandWindows(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		require.NoError(t, shell.Run(testctx.New(), "", []string{"cmd.exe", "/c", "echo", "oi"}, []string{}, false))
	})

	t.Run("cmd failed", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(testctx.New(), "", []string{"cmd.exe", "/c", "exit /b 1"}, []string{}, false),
			`shell: 'cmd.exe /c exit 1': exit status 1: [no output]`,
		)
	})

	t.Run("cmd with output", func(t *testing.T) {
		require.EqualError(
			t,
			shell.Run(testctx.New(), "", []string{"cmd.exe", "/c", "echo something\r\nexit /b 1"}, []string{}, true),
			`shell: 'cmd.exe /c echo something; exit 1': exit status 1: something`,
		)
	})

	t.Run("with env and dir", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, shell.Run(testctx.New(), dir, []string{"cmd.exe", "/c", "copy nul %FOO%"}, []string{"FOO=bar"}, false))
		require.FileExists(t, filepath.Join(dir, "bar"))
	})
}
