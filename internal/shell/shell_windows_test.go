//go:build windows
// +build windows

package shell_test

import (
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
			`shell: 'cmd.exe /c exit /b 1': exit status 1: [no output]`,
		)
	})

	// TODO: more tests for windows
}
