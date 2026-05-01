package nodesea

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/nodedist"
	"github.com/stretchr/testify/require"
)

// stubRunBuildSEA replaces runBuildSEA for the duration of t and
// returns a pointer to the recorded calls so tests can assert on argv
// + the rendered sea-config.json.
type recordedBuildSEA struct {
	NodePath string
	CfgPath  string
	Cfg      map[string]any
}

func stubRunBuildSEA(t *testing.T, behavior func(rec *recordedBuildSEA, tmpOut string) error) *recordedBuildSEA {
	t.Helper()
	rec := &recordedBuildSEA{}
	prev := runBuildSEA
	t.Cleanup(func() { runBuildSEA = prev })
	runBuildSEA = func(_ context.Context, nodePath, cfgPath string) error {
		rec.NodePath = nodePath
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

// stageTargetNode pre-populates the host cache so downloadHost returns
// without hitting the network. Returns the cache root used.
func stageTargetNode(t *testing.T, version string, target nodedist.Target) string {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("TMPDIR", cache)
	cacheDir, err := nodedist.CacheDir()
	require.NoError(t, err)
	hostDir := filepath.Join(cacheDir, version, string(target))
	require.NoError(t, os.MkdirAll(hostDir, 0o755))
	hostPath := filepath.Join(hostDir, target.HostBinaryName())
	require.NoError(t, os.WriteFile(hostPath, []byte("fake target node"), 0o755))
	return cacheDir
}

func TestBuildViaBuildSEA_HappyPath_ELF(t *testing.T) {
	const version = "v22.20.0"
	target := nodedist.Target("linux-x64")
	cacheDir := stageTargetNode(t, version, target)

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

	err := BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: "/fake/build-tool/node",
		Target:        target,
		Version:       version,
		MainJS:        mainPath,
		OutPath:       outPath,
		BuildDir:      buildDir,
	})
	require.NoError(t, err)

	require.FileExists(t, outPath)
	info, err := os.Stat(outPath)
	require.NoError(t, err)
	require.NotZero(t, info.Mode().Perm()&0o111, "output not executable: %v", info.Mode())

	// Recorded sea-config.json round-trip — goreleaser-owned fields
	// must override whatever the user file said.
	require.Equal(t, "/fake/build-tool/node", rec.NodePath)
	require.Equal(t, mainPath, rec.Cfg["main"])
	require.Equal(t, filepath.Join(cacheDir, version, string(target), target.HostBinaryName()), rec.Cfg["executable"])
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
	const version = "v22.20.0"
	target := nodedist.Target("linux-x64")
	stageTargetNode(t, version, target)

	buildDir := t.TempDir()
	mainPath := filepath.Join(buildDir, "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	outPath := filepath.Join(t.TempDir(), "out", "myapp")
	rec := stubRunBuildSEA(t, nil)

	require.NoError(t, BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: "/fake/build-tool/node",
		Target:        target,
		Version:       version,
		MainJS:        mainPath,
		OutPath:       outPath,
		BuildDir:      buildDir,
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
	const version = "v22.20.0"
	target := nodedist.Target("linux-x64")
	stageTargetNode(t, version, target)

	buildDir := t.TempDir()
	mainPath := filepath.Join(buildDir, "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, "sea-config.json"),
		[]byte("{not json"), 0o644))

	outPath := filepath.Join(t.TempDir(), "out", "myapp")
	stubRunBuildSEA(t, nil)

	err := BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: "/fake/build-tool/node",
		Target:        target,
		Version:       version,
		MainJS:        mainPath,
		OutPath:       outPath,
		BuildDir:      buildDir,
	})
	require.ErrorContains(t, err, "parse")
}

func TestBuildViaBuildSEA_AtomicOutput(t *testing.T) {
	const version = "v22.20.0"
	target := nodedist.Target("linux-x64")
	stageTargetNode(t, version, target)

	mainPath := filepath.Join(t.TempDir(), "main.js")
	require.NoError(t, os.WriteFile(mainPath, []byte(`console.log("ok");`), 0o644))

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "myapp")
	require.NoError(t, os.WriteFile(outPath, []byte("pre-existing"), 0o644))

	stubRunBuildSEA(t, func(_ *recordedBuildSEA, tmpOut string) error {
		return fmt.Errorf("simulated --build-sea failure")
	})

	err := BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: "/fake/node",
		Target:        target,
		Version:       version,
		MainJS:        mainPath,
		OutPath:       outPath,
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
	cases := map[string]BuildOptions{
		"missing BuildToolNode": {Target: "linux-x64", Version: "v22.20.0", MainJS: "/m.js", OutPath: "/o"},
		"missing Version":       {BuildToolNode: "/n", Target: "linux-x64", MainJS: "/m.js", OutPath: "/o"},
		"missing MainJS":        {BuildToolNode: "/n", Target: "linux-x64", Version: "v22.20.0", OutPath: "/o"},
		"missing OutPath":       {BuildToolNode: "/n", Target: "linux-x64", Version: "v22.20.0", MainJS: "/m.js"},
		"unsupported target":    {BuildToolNode: "/n", Target: "freebsd-x64", Version: "v22.20.0", MainJS: "/m.js", OutPath: "/o"},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			err := BuildViaBuildSEA(t.Context(), opts)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid BuildOptions")
		})
	}
}

// TestBuildViaBuildSEA_RealNode is the end-to-end smoke test: it runs
// BuildViaBuildSEA against a real Node ≥ v25.5 and execs the produced
// SEA binary on the host. Skipped in -short mode and when the host
// lacks a capable Node.
func TestBuildViaBuildSEA_RealNode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	hostNode, err := exec.LookPath("node")
	if err != nil {
		t.Skip("integration: no `node` in PATH")
	}
	if err := probeBuildSEACapable(t.Context(), hostNode); err != nil {
		t.Skipf("integration: host node lacks --build-sea: %v", err)
	}

	target := nodedist.Target(currentTarget())
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

	require.NoError(t, BuildViaBuildSEA(t.Context(), BuildOptions{
		BuildToolNode: hostNode,
		Target:        target,
		Version:       hostVersion,
		MainJS:        mainPath,
		OutPath:       outPath,
	}))

	require.FileExists(t, outPath)

	// Exec the result and confirm the entrypoint runs.
	cmd := exec.Command(outPath)
	cmd.Env = append(os.Environ(), "NODE_DISABLE_COLORS=1")
	got, err := cmd.CombinedOutput()
	require.NoError(t, err, "exec %s: %s", outPath, got)
	require.Equal(t, "buildsea-ok\n", string(got))
}

// stageTargetNodeWithSHA helper was placeholder-only; removed.
