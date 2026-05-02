package nodesea

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	
	"github.com/stretchr/testify/require"
)

// stubRunBuildSEA replaces runBuildSEA for the duration of t and
// returns a pointer to the recorded calls so tests can assert on argv
// + the rendered sea-config.json.
type recordedBuildSEA struct {
	CfgPath string
	Cfg     map[string]any
}

func stubRunBuildSEA(t *testing.T, behavior func(rec *recordedBuildSEA, tmpOut string) error) *recordedBuildSEA {
	t.Helper()
	rec := &recordedBuildSEA{}
	prev := runBuildSEA
	t.Cleanup(func() { runBuildSEA = prev })
	runBuildSEA = func(_ context.Context, cfgPath string) error {
		rec.CfgPath = cfgPath
		bts, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bts, &rec.Cfg); err != nil {
			return err
		}
		// Simulate Node writing the output binary by creating an empty
		// file at the path the config asks for.
		out, _ := rec.Cfg["output"].(string)
		if behavior != nil {
			return behavior(rec, out)
		}
		return os.WriteFile(out, []byte("fake sea binary"), 0o755)
	}
	return rec
}

// stageTargetNode replaces downloadTargetNode for the duration of t
// and returns the path it will hand back. The returned path points at
// a temp file containing fake host-binary contents, so tests can
// assert against `executable` in the rendered sea-config.json.
func stageTargetNode(t *testing.T, target Target) string {
	t.Helper()
	dir := t.TempDir()
	hostPath := filepath.Join(dir, target.HostBinaryName())
	require.NoError(t, os.WriteFile(hostPath, []byte("fake target node"), 0o755))
	prev := downloadTargetNode
	t.Cleanup(func() { downloadTargetNode = prev })
	downloadTargetNode = func(_ context.Context, _ string, _ Target) (string, error) {
		return hostPath, nil
	}
	return hostPath
}

func TestBuildViaBuildSEA_HappyPath_ELF(t *testing.T) {
	const version = "v25.5.0"
	target := Target("linux-x64")
	hostNode := stageTargetNode(t, target)

	buildDir := t.TempDir()
	mainPath := filepath.Join(buildDir, "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	// User-provided sea-config.json sitting in their project dir.
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

	outPath := filepath.Join(t.TempDir(), "out", "myapp")
	rec := stubRunBuildSEA(t, nil)

	err := buildViaBuildSEA(t.Context(), buildOptions{
		Target:   target,
		Version:  version,
		MainJS:   mainPath,
		OutPath:  outPath,
		BuildDir: buildDir,
	})
	require.NoError(t, err)

	require.FileExists(t, outPath)
	info, err := os.Stat(outPath)
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o111, "output not executable: %v", info.Mode())

	// Recorded sea-config.json round-trip — goreleaser-owned fields
	// must override whatever the user file said.
	require.Equal(t, mainPath, rec.Cfg["main"])
	require.Equal(t, hostNode, rec.Cfg["executable"])
	require.Equal(t, false, rec.Cfg["useCodeCache"])
	require.Equal(t, false, rec.Cfg["useSnapshot"])
	require.Equal(t, false, rec.Cfg["disableExperimentalSEAWarning"], "user-supplied false should be respected")
	require.Equal(t, []any{"--max-old-space-size=4096"}, rec.Cfg["execArgv"])
	require.Equal(t, "module", rec.Cfg["mainFormat"])

	// Relative asset path should be rewritten to absolute (anchored at
	// build dir); absolute path should pass through untouched.
	assets, ok := rec.Cfg["assets"].(map[string]any)
	require.True(t, ok, "assets should be a map: %T", rec.Cfg["assets"])
	require.Equal(t, filepath.Join(buildDir, "assets/icon.png"), assets["icon"])
	require.Equal(t, "/abs/icon.png", assets["abs"])

	// `output` field should point to a sibling tempfile, not the final outPath.
	out, _ := rec.Cfg["output"].(string)
	require.NotEqual(t, outPath, out)
	require.Equal(t, filepath.Dir(outPath), filepath.Dir(filepath.Dir(out)),
		"tempfile %s should live in a scratch dir under %s", out, filepath.Dir(outPath))
}

func TestBuildViaBuildSEA_NoUserSEAConfig(t *testing.T) {
	const version = "v25.5.0"
	target := Target("linux-x64")
	stageTargetNode(t, target)

	buildDir := t.TempDir()
	mainPath := filepath.Join(buildDir, "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	outPath := filepath.Join(t.TempDir(), "out", "myapp")
	rec := stubRunBuildSEA(t, nil)

	require.NoError(t, buildViaBuildSEA(t.Context(), buildOptions{
		Target:   target,
		Version:  version,
		MainJS:   mainPath,
		OutPath:  outPath,
		BuildDir: buildDir,
	}))

	_, hasWarning := rec.Cfg["disableExperimentalSEAWarning"]
	require.False(t, hasWarning, "no user file → goreleaser must not inject disableExperimentalSEAWarning")
	_, hasAssets := rec.Cfg["assets"]
	require.False(t, hasAssets, "no user file → no assets")
	_, hasExec := rec.Cfg["execArgv"]
	require.False(t, hasExec, "no user file → no execArgv")
	_, hasFmt := rec.Cfg["mainFormat"]
	require.False(t, hasFmt, "no user file → no mainFormat")
}

func TestBuildViaBuildSEA_InvalidUserSEAConfig(t *testing.T) {
	const version = "v25.5.0"
	target := Target("linux-x64")
	stageTargetNode(t, target)

	buildDir := t.TempDir()
	mainPath := filepath.Join(buildDir, "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "sea-config.json"),
		[]byte("{not json"), 0o644))

	outPath := filepath.Join(t.TempDir(), "out", "myapp")
	stubRunBuildSEA(t, nil)

	err := buildViaBuildSEA(t.Context(), buildOptions{
		Target:   target,
		Version:  version,
		MainJS:   mainPath,
		OutPath:  outPath,
		BuildDir: buildDir,
	})
	require.ErrorContains(t, err, "parse")
}

func TestBuildViaBuildSEA_AtomicOutput(t *testing.T) {
	const version = "v25.5.0"
	target := Target("linux-x64")
	stageTargetNode(t, target)

	mainPath := filepath.Join(t.TempDir(), "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "myapp")
	require.NoError(t, os.WriteFile(outPath, []byte("pre-existing"), 0o644))

	stubRunBuildSEA(t, func(_ *recordedBuildSEA, tmpOut string) error {
		return fmt.Errorf("simulated --build-sea failure")
	})

	err := buildViaBuildSEA(t.Context(), buildOptions{
		Target:  target,
		Version: version,
		MainJS:  mainPath,
		OutPath: outPath,
	})
	require.Error(t, err)

	// Pre-existing file at outPath must be untouched on failure.
	bts, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.Equal(t, "pre-existing", string(bts))

	// No leftover scratch dirs.
	entries, err := filepath.Glob(filepath.Join(outDir, ".buildsea-*"))
	require.NoError(t, err)
	require.Empty(t, entries, "scratch dir not cleaned up: %v", entries)
}

func TestBuildViaBuildSEA_Validation(t *testing.T) {
	cases := map[string]buildOptions{
		"missing Version":    {Target: "linux-x64", MainJS: "/m.js", OutPath: "/o"},
		"missing MainJS":     {Target: "linux-x64", Version: "v25.5.0", OutPath: "/o"},
		"missing OutPath":    {Target: "linux-x64", Version: "v25.5.0", MainJS: "/m.js"},
		"unsupported target": {Target: "freebsd-x64", Version: "v25.5.0", MainJS: "/m.js", OutPath: "/o"},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			err := buildViaBuildSEA(t.Context(), opts)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid buildOptions")
		})
	}
}

// TestBuildViaBuildSEA_RealNode is the end-to-end smoke test: it runs
// buildViaBuildSEA against the host's `node` and execs the produced
// SEA binary. Skipped in -short mode and when no `node` is on PATH.
func TestBuildViaBuildSEA_RealNode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	hostNode, err := exec.LookPath("node")
	if err != nil {
		t.Skip("integration: no `node` in PATH")
	}

	target := Target(hostTarget())
	if !supportedGoos(target.Goos()) {
		t.Skipf("integration: no SEA injector for host target %s", target)
	}

	out, err := exec.Command(hostNode, "--version").Output()
	require.NoError(t, err)
	hostVersion := string(out)
	hostVersion = hostVersion[:len(hostVersion)-1] // strip trailing newline

	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.js")
	require.NoError(t, os.WriteFile(mainPath,
		[]byte(`process.stdout.write("buildsea-ok\n");`), 0o644))

	outPath := filepath.Join(tmp, "myapp")

	require.NoError(t, buildViaBuildSEA(t.Context(), buildOptions{
		Target:  target,
		Version: hostVersion,
		MainJS:  mainPath,
		OutPath: outPath,
	}))

	require.FileExists(t, outPath)

	// Exec the result and confirm the entrypoint runs.
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
