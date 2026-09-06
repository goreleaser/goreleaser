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

		dir := t.TempDir()
		ready := filepath.Join(dir, "ready")
		release := filepath.Join(dir, "release")
		t.Cleanup(func() {
			require.NoError(t, os.WriteFile(release, nil, 0o600))
		})

		ctx, cancel := context.WithCancel(t.Context())
		errCh := make(chan error, 1)
		go func() {
			errCh <- shell.Run(
				testctx.Wrap(ctx),
				"",
				[]string{"sh", "-c", `sh -c ': > "$READY"; while [ ! -f "$RELEASE" ]; do sleep 1; done' & while :; do sleep 1; done`},
				append(os.Environ(), "READY="+ready, "RELEASE="+release),
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
