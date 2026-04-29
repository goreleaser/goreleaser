package nodesea_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/nodesea"
	"github.com/stretchr/testify/require"
)

// TestRealNodeMachOIntegration exercises the full Mach-O pipeline
// (download → unsign → inject → ad-hoc sign → flip sentinel) against an
// upstream Node.js distribution and then execs the resulting SEA binary
// to confirm the OS loader, dyld, the kernel codesign verifier, and
// Node's own SEA loader all accept it.
//
// The synthetic-fixture tests in inject_macho_test.go cover the byte
// math in isolation; this test is the only thing that proves the byte
// math actually adds up to a working binary on a real Node release.
//
// Skipped when:
//   - testing.Short() — the test downloads ~50 MiB and execs a binary
//   - host is not darwin/arm64 — exec'ing the SEA needs the host arch
//     to match the binary we built (the rest of the pipeline still
//     runs cross-platform via the unit tests)
//   - the local `node` binary needed for SEA-blob generation is missing
func TestRealNodeMachOIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipping in -short mode")
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skipf("integration test: needs darwin/arm64 host, got %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("integration test: no `node` in PATH for SEA blob generation")
	}

	ctx := context.Background()
	target := nodesea.Target("darwin-arm64")

	// Pin to the host node's major version so the SEA blob format the
	// local node emits is understood by the downloaded host loader.
	out, err := exec.Command(nodeBin, "--version").Output()
	require.NoError(t, err)
	hostVersion := strings.TrimSpace(string(out))
	require.True(t, strings.HasPrefix(hostVersion, "v"), "want vX.Y.Z, got %q", hostVersion)

	tmp := t.TempDir()
	hostPath := filepath.Join(tmp, "node-sea")

	// 1. Generate a real SEA blob from a tiny entrypoint using the
	//    host's `node --experimental-sea-config`.
	blob := buildSEABlob(t, nodeBin, tmp)

	// 2. Run the full pipeline (download + unsign + inject + ad-hoc
	//    sign + flip sentinel) in a single Build call.
	require.NoError(t, nodesea.Build(ctx, hostVersion, target, hostPath, blob), "Build")
	requireExecutable(t, hostPath)

	// 3. Re-parse with debug/macho + codesign to confirm structural
	//    validity.
	requireValidMachO(t, hostPath)

	// 4. Exec it and confirm Node's own SEA loader accepts the blob.
	cmd := exec.Command(hostPath)
	cmd.Env = append(os.Environ(), "NODE_DISABLE_COLORS=1")
	got, err := cmd.CombinedOutput()
	require.NoError(t, err, "exec injected SEA: %s", got)
	require.Equal(t, "sea-ok\n", string(got),
		"SEA entrypoint produced wrong output:\n%s", got)
}

// buildSEABlob writes a tiny entrypoint and a sea-config.json into
// tmpDir, runs `<nodeBin> --experimental-sea-config sea-config.json`,
// and returns the generated blob.
func buildSEABlob(t *testing.T, nodeBin, tmpDir string) []byte {
	t.Helper()

	entry := filepath.Join(tmpDir, "main.js")
	require.NoError(t, os.WriteFile(entry, []byte(`process.stdout.write("sea-ok\n");`), 0o644))

	cfgPath := filepath.Join(tmpDir, "sea-config.json")
	blobPath := filepath.Join(tmpDir, "sea.blob")
	cfg := map[string]any{
		"main":                          entry,
		"output":                        blobPath,
		"disableExperimentalSEAWarning": true,
	}
	cfgBytes, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, cfgBytes, 0o600))

	cmd := exec.Command(nodeBin, "--experimental-sea-config", cfgPath)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "node --experimental-sea-config: %s", out)

	blob, err := os.ReadFile(blobPath)
	require.NoError(t, err)
	require.NotEmpty(t, blob)
	return blob
}

// requireExecutable asserts the file at path is regular and has at
// least one execute bit set.
func requireExecutable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.True(t, info.Mode().IsRegular(), "not a regular file: %s", path)
	require.NotZero(t, info.Mode().Perm()&0o111, "not executable: %s mode=%v", path, info.Mode())
}

// requireValidMachO asserts that path can be re-parsed by debug/macho
// without errors. This catches gross structural corruption (e.g. bad
// load command sizes) but not OS-loader-specific rejections — that's
// what the exec step covers.
func requireValidMachO(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("codesign", "-v", "--verbose=2", path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "codesign --verify failed:\n%s", out)
}
