package node

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
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

	// the SEA config lives in a per-target scratch directory: sharing the
	// output directory made concurrent targets overwrite each other's config.
	require.NoFileExists(t, filepath.Join(filepath.Dir(options.Path), "sea-config.json"))
}

// seaTemplate builds the template exactly as Build does, so this test
// fails if Build ever stops feeding target data into the main template.
func seaTemplate(tb testing.TB, target string) *tmpl.Template {
	tb.Helper()
	parsed, err := Default.Parse(target)
	require.NoError(tb, err)
	options := api.Options{
		Name:   "proj",
		Path:   filepath.Join("dist", "proj_"+target, "proj"),
		Target: parsed,
	}
	ctx := testctx.WrapWithCfg(tb.Context(), config.Project{ProjectName: "proj"})
	return tmpl.New(ctx).
		WithBuildOptions(options).
		WithArtifact(&artifact.Artifact{
			Type:   artifact.Binary,
			Path:   options.Path,
			Name:   options.Name,
			Goos:   parsed.(Target).Goos(),
			Goarch: parsed.(Target).Goarch(),
			Target: target,
			Extra: map[string]any{
				artifact.ExtraBinary:  "proj",
				artifact.ExtraID:      "default",
				artifact.ExtraBuilder: "node",
			},
		})
}

func TestCreateSEAConfigRendersMain(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		main   string
		want   string
	}{
		"literal": {"darwin-arm64", "index.js", "index.js"},
		"target":  {"darwin-arm64", "build/{{ .Target }}.js", "build/darwin-arm64.js"},
		"os":      {"darwin-arm64", "build/{{ .Os }}.js", "build/darwin.js"},
		"arch":    {"darwin-arm64", "build/{{ .Arch }}.js", "build/arm64.js"},
		// .Os and .Arch carry the Go names, not the nodejs.org ones.
		"go names": {"win-x64", "build/{{ .Os }}-{{ .Arch }}.js", "build/windows-amd64.js"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			mainPath := filepath.Join(dir, filepath.FromSlash(tc.want))
			require.NoError(t, os.MkdirAll(filepath.Dir(mainPath), 0o755))
			require.NoError(t, os.WriteFile(mainPath, []byte("console.log('ok')\n"), 0o644))

			cfgPath := filepath.Join(t.TempDir(), "sea-config.json")
			require.NoError(t, createSEAConfig(
				seaTemplate(t, tc.target),
				config.Build{ID: "default", Dir: dir, Main: tc.main},
				cfgPath,
				filepath.Join(dir, "node"),
				filepath.Join(dir, "proj"),
			))

			bts, err := os.ReadFile(cfgPath)
			require.NoError(t, err)
			var cfg struct {
				Main string `json:"main"`
			}
			require.NoError(t, json.Unmarshal(bts, &cfg))
			require.Equal(t, mainPath, cfg.Main)
		})
	}
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
