package zig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestParse(t *testing.T) {
	for target, dst := range map[string]Target{
		"x86_64-linux": {
			Target: "x86_64-linux",
			Os:     "linux",
			Arch:   "amd64",
		},
		"x86_64-linux-gnu": {
			Target: "x86_64-linux-gnu",
			Os:     "linux",
			Arch:   "amd64",
			Abi:    "gnu",
		},
		"aarch64-linux-gnu": {
			Target: "aarch64-linux-gnu",
			Os:     "linux",
			Arch:   "arm64",
			Abi:    "gnu",
		},
		"aarch64-linux": {
			Target: "aarch64-linux",
			Os:     "linux",
			Arch:   "arm64",
		},
		"aarch64-macos": {
			Target: "aarch64-macos",
			Os:     "darwin",
			Arch:   "arm64",
		},
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
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "zig",
			Command: "build",
			Dir:     ".",
			Targets: defaultTargets(),
			Flags:   []string{"-Doptimize=ReleaseSafe"},
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})

	t.Run("empty target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{""},
		})
		require.ErrorContains(t, err, "invalid target")
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Main: "something",
		})
		require.Error(t, err)
	})
}

func TestBuildCopiesCompilerBinaryBasename(t *testing.T) {
	for name, tt := range map[string]struct {
		binary string
	}{
		"unwrapped": {binary: "app"},
		"wrapped":   {binary: filepath.Join("bin", "app")},
	} {
		t.Run(name, func(t *testing.T) {
			testlib.Mktmp(t)
			createFakeZigBuild(t, filepath.Join("zig-out", "aarch64-macos", "bin", "app"), "built by zig")

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: "app",
			})
			build, err := Default.WithDefaults(config.Build{
				ID:      "default",
				Dir:     ".",
				Targets: []string{"aarch64-macos"},
			})
			require.NoError(t, err)

			target, err := Default.Parse("aarch64-macos")
			require.NoError(t, err)
			options := api.Options{
				Name:   tt.binary,
				Path:   filepath.Join("dist", "default_aarch64-macos", tt.binary),
				Target: target,
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(options.Path), 0o755))

			require.NoError(t, Default.Build(ctx, build, options))

			got, err := os.ReadFile(options.Path)
			require.NoError(t, err)
			require.Equal(t, "built by zig", string(got))

			bins := ctx.Artifacts.List()
			require.Len(t, bins, 1)
			require.Equal(t, tt.binary, bins[0].Name)
			require.Equal(t, filepath.ToSlash(options.Path), bins[0].Path)
			require.Equal(t, "app", bins[0].Extra[artifact.ExtraBinary])
		})
	}
}

func createFakeZigBuild(tb testing.TB, output, contents string) {
	tb.Helper()
	dir := tb.TempDir()
	name := "zig"
	script := fmt.Sprintf(`#!/bin/sh
mkdir -p %q
printf '%%s' %q > %q
`, filepath.ToSlash(filepath.Dir(output)), contents, filepath.ToSlash(output))
	if runtime.GOOS == "windows" {
		name += ".bat"
		output = filepath.Clean(output)
		outputDir := filepath.Dir(output)
		script = fmt.Sprintf(
			"@echo off\r\nif not exist \"%s\" mkdir \"%s\"\r\n> \"%s\" <nul set /p dummy=%s\r\nexit /b 0\r\n",
			outputDir,
			outputDir,
			output,
			contents,
		)
	}
	require.NoError(tb, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "zig")

	folder := t.TempDir()
	require.NoError(t, gio.Copy("testdata/proj", filepath.Join(folder, "proj")))
	t.Chdir(folder)
	// the local cache stays per-test so build outputs cannot collide; the
	// global cache is shared, see testlib.SharedZigCache.
	t.Setenv("ZIG_LOCAL_CACHE_DIR", filepath.Join(folder, ".zig-cache"))
	testlib.SharedZigCache(t)
	folder = filepath.Join(folder, "proj")

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Env: []string{
			"OPTIMIZE_FOR=Debug",
		},
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          "./proj/",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Flags:        []string{"-Doptimize={{.Env.OPTIM}}"},
				Env: []string{
					"OPTIM={{.Env.OPTIMIZE_FOR}}",
				},
			},
		},
	})

	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)

	options := api.Options{
		Name:   "proj",
		Path:   filepath.Join("dist", "proj-aarch64-macos", "proj"),
		Target: nil,
	}
	options.Target, err = Default.Parse("aarch64-macos")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(options.Path), 0o755)) // this happens on internal/pipe/build/ when in prod

	require.NoError(t, Default.Build(ctx, build, options))

	list := ctx.Artifacts
	require.NoError(t, list.Visit(func(a *artifact.Artifact) error {
		s, err := filepath.Rel(folder, a.Path)
		if err == nil {
			a.Path = s
		}
		return nil
	}))

	bins := list.List()
	require.Len(t, bins, 1)

	bin := bins[0]
	require.Equal(t, artifact.Artifact{
		Name:   "proj",
		Path:   filepath.ToSlash(options.Path),
		Goos:   "darwin",
		Goarch: "arm64",
		Target: "aarch64-macos",
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:  "proj",
			artifact.ExtraBuilder: "zig",
			artifact.ExtraExt:     "",
			artifact.ExtraID:      "default",
			keyAbi:                "",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}
