package shell_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/shell"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestRunCommand(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		require.NoError(t, shell.Run(
			testctx.Wrap(t.Context()),
			"",
			strings.Fields(testlib.Echo("oi")),
			[]string{},
			false,
		))
	})

	t.Run("empty command", func(t *testing.T) {
		require.NoError(t, shell.Run(
			testctx.Wrap(t.Context()),
			"",
			[]string{},
			[]string{},
			false,
		))
	})

	t.Run("cmd failed", func(t *testing.T) {
		require.Error(t, shell.Run(
			testctx.Wrap(t.Context()),
			"",
			strings.Fields(testlib.Exit(1)),
			[]string{},
			false,
		))
	})

	t.Run("cmd with output", func(t *testing.T) {
		testlib.SkipIfWindows(t, "what would be a similar behavior in windows?")
		err := shell.Run(
			testctx.Wrap(t.Context()),
			".",
			[]string{"sh", "-c", `echo something; exit 1`},
			[]string{},
			true,
		)
		require.EqualError(
			t, err,
			`exit status 1`,
		)
	})

	t.Run("cancellation with descendant-held output pipe", func(t *testing.T) {
		testlib.SkipIfWindows(t, "uses a unix shell")

		ready := filepath.Join(t.TempDir(), "ready")

		ctx, cancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- shell.Run(
				testctx.Wrap(ctx),
				"",
				// the descendant keeps stdout and stderr open for much longer
				// than WaitDelay, and stops by itself so the test leaks nothing.
				[]string{"sh", "-c", `sh -c ': > "$READY"; sleep 30' & while :; do sleep 1; done`},
				append(os.Environ(), "READY="+ready),
				false,
			)
		}()

		require.Eventually(t, func() bool {
			_, err := os.Stat(ready)
			return err == nil
		}, 3*time.Second, 10*time.Millisecond, "descendant did not start")
		cancel()

		select {
		case err := <-errCh:
			require.Error(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("command did not return while descendant held stdout and stderr open")
		}
	})

	t.Run("success with descendant-held output pipe", func(t *testing.T) {
		testlib.SkipIfWindows(t, "uses a unix shell")

		errCh := make(chan error, 1)
		go func() {
			errCh <- shell.Run(
				testctx.Wrap(t.Context()),
				"",
				// exits 0 while a background job keeps stdout and stderr open.
				[]string{"sh", "-c", `sleep 30 & echo done`},
				os.Environ(),
				false,
			)
		}()

		select {
		case err := <-errCh:
			require.NoError(t, err, "a successful command must not fail because a descendant held its pipes")
		case <-time.After(5 * time.Second):
			t.Fatal("command did not return while descendant held stdout and stderr open")
		}
	})

	t.Run("with env and dir", func(t *testing.T) {
		testlib.SkipIfWindows(t, "what would be a similar behavior in windows?")
		dir := t.TempDir()
		touch, err := exec.LookPath("touch")
		require.NoError(t, err)
		err = shell.Run(
			testctx.Wrap(t.Context()),
			dir,
			[]string{"sh", "-c", touch + " $FOO"},
			[]string{"FOO=bar"},
			false,
		)
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(dir, "bar"))
	})
}

func TestRunRedactsDebugCommand(t *testing.T) {
	testlib.SkipIfWindows(t, "uses sh")

	var logs bytes.Buffer
	previousLog := log.Log
	log.Log = log.New(&logs)
	log.SetLevel(log.DebugLevel)
	t.Cleanup(func() {
		log.Log = previousLog
	})

	const secret = "key123key123"
	err := shell.Run(
		testctx.Wrap(t.Context()),
		"",
		[]string{
			"sh",
			"-c",
			`test "$API_KEY" = "key123key123" && echo "$API_KEY"`,
		},
		[]string{"API_KEY=" + secret},
		true,
	)
	require.NoError(t, err)
	require.NotContains(t, logs.String(), secret)
	require.Contains(t, logs.String(), "$API_KEY")
}
