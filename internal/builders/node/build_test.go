package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"node"}, Default.Dependencies())
}

func TestAllowConcurrentBuilds(t *testing.T) {
	require.True(t, Default.AllowConcurrentBuilds())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"linux-x64":    {Target: "linux-x64", Os: "linux", Arch: "x64"},
		"darwin-arm64": {Target: "darwin-arm64", Os: "darwin", Arch: "arm64"},
		"win-arm64":    {Target: "win-arm64", Os: "win", Arch: "arm64"},
	} {
		t.Run(target, func(t *testing.T) {
			got, err := Default.Parse(target)
			require.NoError(t, err)
			require.IsType(t, Target{}, got)
			require.Equal(t, dst, got.(Target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("linux")
		require.Error(t, err)
	})
	t.Run("unknown target", func(t *testing.T) {
		_, err := Default.Parse("plan9-amd64")
		require.Error(t, err)
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, ".", build.Dir)
		require.Equal(t, "index.js", build.Main)
		require.Equal(t, "node", build.Tool)
		require.ElementsMatch(t, defaultTargets(), build.Targets)
	})

	t.Run("respects main override", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{Main: "src/cli.mjs"})
		require.NoError(t, err)
		require.Equal(t, "src/cli.mjs", build.Main)
	})

	t.Run("respects tool override", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{Tool: "/opt/node/bin/node"})
		require.NoError(t, err)
		require.Equal(t, "/opt/node/bin/node", build.Tool)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"plan9-amd64"},
		})
		require.Error(t, err)
	})

	t.Run("rejects command", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{Command: "build"})
		require.ErrorContains(t, err, "command is not supported")
	})

	t.Run("rejects flags", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Flags: []string{"--x"},
		})
		require.ErrorContains(t, err, "flags is not supported")
	})
}

func TestResolveVersionStringRejectsUnsupportedSEARelease(t *testing.T) {
	_, err := resolveVersionString("20.0.0")
	require.ErrorContains(t, err, ">= v25.5.0")
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "node")

	target := "darwin-arm64"
	createFakeNodeAlias(t, "node-"+target)

	out, err := exec.Command("node", "--version").Output()
	require.NoError(t, err)
	hostVersion := strings.TrimSpace(string(out))

	testlib.Mktmp(t)
	require.NoError(t, os.WriteFile("index.js",
		[]byte(`process.stdout.write("buildsea-ok\n");`), 0o644))
	require.NoError(t, os.WriteFile("package.json",
		[]byte(`{"engines":{"node":"`+hostVersion+`"}}`), 0o644))
	require.NoError(t, os.WriteFile("sea-config.json",
		[]byte(`{"disableExperimentalSEAWarning": true}`), 0o644))

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				Tool:         "node-{{ .Target }}",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
			},
		},
	})

	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)

	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj_"+target, "proj"),
	}
	options.Target, err = Default.Parse(target)
	require.NoError(t, err)

	require.NoError(t, Default.Build(ctx, build, options))

	bins := ctx.Artifacts.List()
	require.Len(t, bins, 1)
	bin := bins[0]
	require.Equal(t, filepath.ToSlash(options.Path), bin.Path)

	fi, err := os.Stat(filepath.FromSlash(bin.Path))
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}

func TestBuildUsesPerTargetSEAConfig(t *testing.T) {
	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	require.NoError(t, os.Mkdir(projectDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "index.js"), []byte("console.log('ok')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(`{"engines":{"node":"25.5.0"}}`), 0o644))

	prevDownloadNodeBinary := downloadNodeBinary
	downloadNodeBinary = func(_ context.Context, _ string, target string, destDir string) (string, error) {
		targetNode := filepath.Join(destDir, "node-"+target)
		return targetNode, os.WriteFile(targetNode, []byte(target), 0o755)
	}
	t.Cleanup(func() { downloadNodeBinary = prevDownloadNodeBinary })

	var (
		mu      sync.Mutex
		ready   int
		release = make(chan struct{})
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		ready++
		if ready == 2 {
			close(release)
		}
		mu.Unlock()
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	recordDir := filepath.Join(tmp, "records")
	require.NoError(t, os.Mkdir(recordDir, 0o755))
	t.Setenv("NODE_BUILD_HELPER_BARRIER_URL", server.URL+"/ready")
	t.Setenv("NODE_BUILD_HELPER_RECORD_DIR", recordDir)

	build := config.Build{
		ID:   "default",
		Dir:  projectDir,
		Main: "index.js",
		Tool: createFakeNodeBuildHelper(t),
	}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        filepath.Join(tmp, "dist"),
		ProjectName: "proj",
	})

	targets := []string{"linux-x64", "linux-arm64"}
	outputs := map[string]string{}
	var wg sync.WaitGroup
	errs := make(chan error, len(targets))
	for _, target := range targets {
		output := filepath.Join(tmp, "dist", "proj-"+target)
		outputs[target] = output
		wg.Go(func() {
			parsed, err := Default.Parse(target)
			if err != nil {
				errs <- err
				return
			}
			errs <- Default.Build(ctx, build, api.Options{
				Name:   "proj-" + target,
				Path:   output,
				Target: parsed,
			})
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	for _, output := range outputs {
		require.FileExists(t, output)
	}

	records, err := os.ReadDir(recordDir)
	require.NoError(t, err)
	require.Len(t, records, 2)

	configs := map[string]bool{}
	executables := map[string]bool{}
	recordedOutputs := map[string]bool{}
	for _, record := range records {
		bts, err := os.ReadFile(filepath.Join(recordDir, record.Name()))
		require.NoError(t, err)
		parts := strings.Split(strings.TrimSpace(string(bts)), "\n")
		require.Len(t, parts, 3)
		configs[parts[0]] = true
		executables[parts[1]] = true
		recordedOutputs[parts[2]] = true
	}

	require.Len(t, configs, 2)
	require.Len(t, executables, 2)
	require.Equal(t, map[string]bool{
		outputs["linux-x64"]:   true,
		outputs["linux-arm64"]: true,
	}, recordedOutputs)
}

func TestBuildRejectsUnsupportedHostNode(t *testing.T) {
	testlib.Mktmp(t)
	require.NoError(t, os.WriteFile("index.js", []byte(`process.stdout.write("nope\n");`), 0o644))
	require.NoError(t, os.WriteFile("package.json", []byte(`{"engines":{"node":"0.0.1"}}`), 0o644))
	createFakeNodeVersion(t, "node-v20", "v20.0.0")

	target, err := Default.Parse("linux-x64")
	require.NoError(t, err)

	err = Default.Build(
		testctx.Wrap(t.Context()),
		config.Build{Dir: ".", Main: "index.js", Tool: "node-v20"},
		api.Options{
			Name:   "proj",
			Path:   filepath.Join("dist", "proj"),
			Target: target,
		},
	)

	require.ErrorContains(t, err, "host node")
	require.ErrorContains(t, err, ">= v25.5.0")
}

func createFakeNodeAlias(tb testing.TB, name string) {
	tb.Helper()
	node, err := exec.LookPath("node")
	require.NoError(tb, err)
	createFakeExecutable(
		tb, name,
		fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", node),
		fmt.Sprintf("@echo off\n%q %%*\n", node),
	)
}

func createFakeNodeBuildHelper(tb testing.TB) string {
	tb.Helper()
	exe, err := os.Executable()
	require.NoError(tb, err)
	const name = "node-build-helper"
	createFakeExecutable(
		tb, name,
		fmt.Sprintf("#!/bin/sh\nGO_WANT_NODE_BUILD_HELPER_PROCESS=1 exec %q -test.run=TestNodeBuildHelperProcess -- \"$@\"\n", exe),
		fmt.Sprintf("@echo off\nset GO_WANT_NODE_BUILD_HELPER_PROCESS=1\n%q -test.run=TestNodeBuildHelperProcess -- %%*\n", exe),
	)
	return name
}

func TestNodeBuildHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_NODE_BUILD_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		args = args[1:]
	}

	switch {
	case len(args) == 1 && args[0] == "--version":
		fmt.Println("v25.5.0")
	case len(args) == 2 && args[0] == "--build-sea":
		runNodeBuildHelper(args[1])
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unexpected node helper args: %q\n", args)
		os.Exit(2)
	}
	os.Exit(0)
}

func runNodeBuildHelper(cfgPath string) {
	resp, err := http.Get(os.Getenv("NODE_BUILD_HELPER_BARRIER_URL"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "barrier: %v\n", err)
		os.Exit(1)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		_, _ = fmt.Fprintf(os.Stderr, "barrier status: %s\n", resp.Status)
		os.Exit(1)
	}

	bts, err := os.ReadFile(cfgPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read sea config: %v\n", err)
		os.Exit(1)
	}
	var cfg struct {
		Executable string `json:"executable"`
		Output     string `json:"output"`
	}
	if err := json.Unmarshal(bts, &cfg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse sea config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(cfg.Output, []byte(cfg.Executable), 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
	record := strings.Join([]string{cfgPath, cfg.Executable, cfg.Output}, "\n")
	recordPath := filepath.Join(os.Getenv("NODE_BUILD_HELPER_RECORD_DIR"), fmt.Sprintf("%d", os.Getpid()))
	if err := os.WriteFile(recordPath, []byte(record), 0o600); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write record: %v\n", err)
		os.Exit(1)
	}
}

func createFakeNodeVersion(tb testing.TB, name, version string) {
	tb.Helper()
	createFakeExecutable(
		tb, name,
		fmt.Sprintf("#!/bin/sh\necho %s\n", version),
		fmt.Sprintf("@echo off\necho %s\n", version),
	)
}

func createFakeExecutable(tb testing.TB, name, unix, windows string) {
	tb.Helper()
	dir := tb.TempDir()
	if runtime.GOOS == "windows" {
		name += ".bat"
		unix = windows
	}
	require.NoError(tb, os.WriteFile(filepath.Join(dir, name), []byte(unix), 0o755))
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
