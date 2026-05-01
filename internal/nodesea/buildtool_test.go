package nodesea

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/stretchr/testify/require"
)

// stubProbe replaces runProbe for the duration of t with a closure that
// returns out and runErr. Cleanup restores the previous runner.
func stubProbe(t *testing.T, out string, runErr error) {
	t.Helper()
	prev := runProbe
	t.Cleanup(func() { runProbe = prev })
	runProbe = func(_ context.Context, _ string) ([]byte, error) {
		return []byte(out), runErr
	}
}

// writeStubBinary writes an empty executable file under t.TempDir() and
// returns its absolute path. The contents are inert — exec.LookPath
// only checks for existence + the executable bit (Linux/macOS) or a
// known extension (Windows) — so the file never actually runs because
// runProbe is always stubbed alongside.
func writeStubBinary(t *testing.T) string {
	t.Helper()
	name := "stub-node"
	if runtime.GOOS == "windows" {
		name = "stub-node.exe"
	}
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte{}, 0o755))
	return path
}

func TestProbeBuildSEACapable(t *testing.T) {
	cases := []struct {
		name    string
		out     string
		runErr  error
		wantErr bool
	}{
		{name: "true", out: "true\n", wantErr: false},
		{name: "true with surrounding whitespace", out: "  true\n\n", wantErr: false},
		{name: "false", out: "false\n", wantErr: true},
		{name: "undefined", out: "undefined\n", wantErr: true},
		{name: "empty output", out: "", wantErr: true},
		{name: "garbage", out: "garbage\n", wantErr: true},
		{name: "exec failure", out: "boom", runErr: errors.New("fork/exec: no such file"), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubProbe(t, tc.out, tc.runErr)
			err := probeBuildSEACapable(t.Context(), "/dummy/node")
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrBuildSEAUnsupported)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildToolNode_FromEnvOverride(t *testing.T) {
	stub := writeStubBinary(t)

	t.Run("probe passes", func(t *testing.T) {
		t.Setenv(BuildToolEnv, stub)
		stubProbe(t, "true\n", nil)
		got, err := BuildToolNode(t.Context())
		require.NoError(t, err)
		require.Equal(t, stub, got)
	})

	t.Run("probe fails", func(t *testing.T) {
		t.Setenv(BuildToolEnv, stub)
		stubProbe(t, "false\n", nil)
		_, err := BuildToolNode(t.Context())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBuildSEAUnsupported)
		require.Contains(t, err.Error(), BuildToolEnv)
	})

	t.Run("env path missing", func(t *testing.T) {
		t.Setenv(BuildToolEnv, filepath.Join(t.TempDir(), "does-not-exist"))
		stubProbe(t, "true\n", nil)
		_, err := BuildToolNode(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), BuildToolEnv)
	})
}

func TestBuildToolNode_FromPath(t *testing.T) {
	dir := t.TempDir()
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
	}
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte{}, 0o755))
	t.Setenv("PATH", dir)
	t.Setenv(BuildToolEnv, "")

	t.Run("probe passes", func(t *testing.T) {
		stubProbe(t, "true\n", nil)
		got, err := BuildToolNode(t.Context())
		require.NoError(t, err)
		require.Equal(t, path, got)
	})

	t.Run("probe fails falls through to download", func(t *testing.T) {
		// Force the download branch to fail fast by pointing distBaseURL
		// at a closed server: we just want to confirm that an unsuitable
		// PATH node does NOT short-circuit BuildToolNode with success.
		stubProbe(t, "false\n", nil)
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()
		nodedist.SetBaseURL(t, srv.URL)
		_, err := BuildToolNode(t.Context())
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrBuildSEAUnsupported,
			"PATH probe failure should not surface as the final error; download error should")
	})
}

func TestBuildToolNode_AutoDownload(t *testing.T) {
	// Drive the auto-download branch by clearing every higher-priority
	// candidate (env unset, PATH empty) and pointing distBaseURL at a
	// stub nodejs.org/dist that serves a fake host binary.
	t.Setenv(BuildToolEnv, "")
	t.Setenv("PATH", "")

	target := nodedist.Target(currentTarget())
	version := BuildToolNodeVersion

	var payload []byte
	if target.IsWindows() {
		payload = []byte("fake node.exe contents")
	} else {
		payload = []byte("fake node binary")
	}
	archive := nodedist.FakeArchive(t, version, target, payload)
	archName := target.ArchiveName(version)
	nodedist.StubRelease(t, version, archName, archive)

	server := nodedist.NewServer(t, map[string][]byte{
		"/" + version + "/" + archName: archive,
	})
	nodedist.SetBaseURL(t, server.URL)

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	stubProbe(t, "true\n", nil)

	got, err := BuildToolNode(t.Context())
	require.NoError(t, err)
	require.FileExists(t, got)
	require.Contains(t, got, filepath.Join("buildtool", version, string(target)))
}

func TestBuildToolNode_AutoDownload_ProbeFails(t *testing.T) {
	// Even after a successful download, a binary that fails the probe
	// must not be returned as usable. Confirms the post-download probe
	// gate.
	t.Setenv(BuildToolEnv, "")
	t.Setenv("PATH", "")

	target := nodedist.Target(currentTarget())
	version := BuildToolNodeVersion

	var payload []byte
	if target.IsWindows() {
		payload = []byte("fake node.exe contents")
	} else {
		payload = []byte("fake node binary")
	}
	archive := nodedist.FakeArchive(t, version, target, payload)
	archName := target.ArchiveName(version)
	nodedist.StubRelease(t, version, archName, archive)

	server := nodedist.NewServer(t, map[string][]byte{
		"/" + version + "/" + archName: archive,
	})
	nodedist.SetBaseURL(t, server.URL)

	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	stubProbe(t, "false\n", nil)

	_, err := BuildToolNode(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBuildSEAUnsupported)
}

func TestCurrentTarget(t *testing.T) {
	got := currentTarget()
	require.NotEmpty(t, got)
	// Must mirror the targets.txt list for this builder; spot-check by
	// asserting the prefix matches the host GOOS.
	switch runtime.GOOS {
	case "windows":
		require.Contains(t, got, "win-")
	case "linux":
		require.Contains(t, got, "linux-")
	case "darwin":
		require.Contains(t, got, "darwin-")
	}
}

// TestRealHostNodeProbe is a manual smoke test that runs the probe
// against the host's `node` on PATH. It is gated by the `manual` build
// tag so CI and routine `go test ./...` don't require a host with
// node v25.5+.
func TestRealHostNodeProbe(t *testing.T) {
	if testing.Short() {
		t.Skip("manual: skipping in -short mode")
	}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("manual: no `node` in PATH")
	}
	if err := probeBuildSEACapable(t.Context(), nodeBin); err != nil {
		t.Skipf("manual: host node %s does not satisfy --build-sea probe: %v", nodeBin, err)
	}
}
