package node

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// stageBuildDir creates a project directory with package.json declaring
// engines.node = version and a main.js entrypoint. Returns (buildDir,
// mainPath).
func stageBuildDir(t *testing.T, version string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	main := filepath.Join(dir, "main.js")
	require.NoError(t, os.WriteFile(main, []byte(`console.log("ok");`), 0o644))
	pkg := `{"engines":{"node":"` + version + `"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644))
	return dir, main
}

func mustParseTarget(t *testing.T, s string) Target {
	t.Helper()
	tt, ok := parseTarget(s)
	require.True(t, ok, "parseTarget(%q)", s)
	return tt
}

func TestBuildSEAConfigJSON_MergesUserConfig(t *testing.T) {
	buildDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "sea-config.json"), []byte(`{
  "assets": {"icon": "assets/icon.png", "abs": "/abs/icon.png"},
  "execArgv": ["--max-old-space-size=4096"],
  "disableExperimentalSEAWarning": false,
  "mainFormat": "module",
  "main": "ignored.js",
  "output": "ignored",
  "executable": "ignored",
  "useCodeCache": true,
  "useSnapshot": true
}`), 0o644))

	cfg, err := buildSEAConfigJSON(buildDir, "/abs/main.js", "/abs/node", "/abs/out/myapp")
	require.NoError(t, err)

	// Goreleaser-owned fields must override whatever the user file said.
	require.Equal(t, "/abs/main.js", cfg["main"])
	require.Equal(t, "/abs/node", cfg["executable"])
	require.Equal(t, "/abs/out/myapp", cfg["output"])
	require.Equal(t, false, cfg["useCodeCache"])
	require.Equal(t, false, cfg["useSnapshot"])

	// User-tunable fields pass through.
	require.Equal(t, false, cfg["disableExperimentalSEAWarning"])
	require.Equal(t, []any{"--max-old-space-size=4096"}, cfg["execArgv"])
	require.Equal(t, "module", cfg["mainFormat"])

	// Relative asset path rewritten to absolute (anchored at build dir);
	// absolute path passes through untouched.
	assets, ok := cfg["assets"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, filepath.Join(buildDir, "assets/icon.png"), assets["icon"])
	require.Equal(t, "/abs/icon.png", assets["abs"])
}

func TestBuildSEAConfigJSON_NoUserFile(t *testing.T) {
	cfg, err := buildSEAConfigJSON(t.TempDir(), "/m.js", "/n", "/o")
	require.NoError(t, err)

	for _, k := range []string{"disableExperimentalSEAWarning", "assets", "execArgv", "mainFormat"} {
		_, has := cfg[k]
		require.False(t, has, "no user file → must not inject %q", k)
	}
}

func TestBuildSEAConfigJSON_InvalidUserFile(t *testing.T) {
	buildDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "sea-config.json"),
		[]byte("{not json"), 0o644))
	_, err := buildSEAConfigJSON(buildDir, "/m.js", "/n", "/o")
	require.ErrorContains(t, err, "parse")
}

func TestBuildSEA_UnsupportedTarget(t *testing.T) {
	err := buildSEA(t.Context(), Target{Target: "freebsd-x64", Os: "freebsd", Arch: "x64"}, t.TempDir(), "/m.js", "/o")
	require.ErrorContains(t, err, "unsupported target")
}

// TestBuildSEA_RealNode is the end-to-end smoke test: it runs buildSEA
// against the host's `node` and execs the produced SEA binary.
// Skipped in -short mode and when no `node` is on PATH.
func TestBuildSEA_RealNode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	hostNode, err := exec.LookPath("node")
	if err != nil {
		t.Skip("integration: no `node` in PATH")
	}

	target, ok := parseTarget(hostTarget())
	if !ok {
		t.Skipf("integration: no SEA injector for host target %s", hostTarget())
	}

	out, err := exec.Command(hostNode, "--version").Output()
	require.NoError(t, err)
	hostVersion := string(out)
	hostVersion = hostVersion[:len(hostVersion)-1] // strip trailing newline

	buildDir, mainPath := stageBuildDir(t, hostVersion)
	require.NoError(t, os.WriteFile(mainPath,
		[]byte(`process.stdout.write("buildsea-ok\n");`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "sea-config.json"),
		[]byte(`{"disableExperimentalSEAWarning": true}`), 0o644))

	outPath := filepath.Join(t.TempDir(), "myapp")
	require.NoError(t, buildSEA(t.Context(), target, buildDir, mainPath, outPath))
	require.FileExists(t, outPath)

	cmd := exec.Command(outPath)
	cmd.Env = append(os.Environ(), "NODE_DISABLE_COLORS=1")
	got, err := cmd.CombinedOutput()
	require.NoError(t, err, "exec %s: %s", outPath, got)
	require.Equal(t, "buildsea-ok\n", string(got))
}

// hostTarget returns the nodejs.org/dist target identifier matching the
// machine running the test.
func hostTarget() string {
	osName := runtime.GOOS
	if osName == "windows" {
		osName = "win"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return osName + "-" + arch
}
