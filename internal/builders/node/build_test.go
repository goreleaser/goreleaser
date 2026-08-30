package node

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func TestExtractFromTarGz(t *testing.T) {
	const entry = "node-v25.5.0-linux-x64/bin/node"
	// Larger than deflate's 32KiB history window, and compressible, so
	// the fixture spans a window boundary and more than one block —
	// the decoder state a replacement inflate could get wrong. A
	// three-entry toy archive would fit in a single block and prove
	// nothing.
	payload := strings.Repeat("#!/bin/sh\nexec node \"$@\"\n", 2000)
	archive := writeTarGz(t, map[string]string{
		"node-v25.5.0-linux-x64/README.md": "readme",
		entry:                              payload,
	})

	t.Run("requested entry", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "node")
		require.NoError(t, extractFromTarGz(archive, entry, dst))
		bts, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, payload, string(bts))
	})

	t.Run("missing entry", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "node")
		err := extractFromTarGz(archive, "node-v25.5.0-linux-x64/bin/nope", dst)
		require.ErrorContains(t, err, "not found in")
		require.NoFileExists(t, dst)
	})
}

// writeTarGz builds a gzipped tar from entries and returns its path.
// Written with compress/gzip so the test also covers reading a
// stdlib-produced stream back with klauspost's decompressor.
func writeTarGz(tb testing.TB, entries map[string]string) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "archive.tar.gz")
	f, err := os.Create(path)
	require.NoError(tb, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		require.NoError(tb, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(body)),
		}))
		_, err := tw.Write([]byte(body))
		require.NoError(tb, err)
	}
	require.NoError(tb, tw.Close())
	require.NoError(tb, gz.Close())
	require.NoError(tb, f.Close())
	return path
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
