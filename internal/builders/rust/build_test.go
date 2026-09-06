package rust

import (
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
	api "github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestAllowConcurrentBuilds(t *testing.T) {
	require.False(t, Default.AllowConcurrentBuilds())
}

func TestDependencies(t *testing.T) {
	require.NotEmpty(t, Default.Dependencies())
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "cargo",
			Command: "zigbuild",
			Dir:     ".",
			Targets: defaultTargets(),
			Flags:   []string{"--release"},
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

func TestCustomGlibc(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-unknown-linux-gnu.2.17"},
		})
		require.NoError(t, err)
	})
	t.Run("valid-gnueabihf", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"armv7-unknown-linux-gnueabihf.2.17"},
		})
		require.NoError(t, err)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-unknown-linux-musl.2.17"},
		})
		require.ErrorContains(t, err, "invalid target")
	})
	t.Run("invalid-gnullvm", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"aarch64-pc-windows-gnullvm.2.17"},
		})
		require.ErrorContains(t, err, "invalid target")
	})
}

func TestPrepareUsesBuildContext(t *testing.T) {
	folder := testlib.Mktmp(t)
	target := "aarch64-unknown-linux-gnu.2.17"

	for name, tt := range map[string]struct {
		projectEnv    []string
		buildEnv      []string
		wantToolchain string
	}{
		"nested rust-toolchain": {},
		"project environment": {
			projectEnv:    []string{"RUSTUP_TOOLCHAIN=1.95.0"},
			wantToolchain: "1.95.0",
		},
		"build environment": {
			projectEnv:    []string{"TOOLCHAIN=1.95.0"},
			buildEnv:      []string{"RUSTUP_TOOLCHAIN={{.Env.TOOLCHAIN}}"},
			wantToolchain: "1.95.0",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("RUSTUP_TOOLCHAIN", "")

			dir := filepath.Join("nested", name)
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile("rust-toolchain.toml", []byte("stable\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "rust-toolchain.toml"), []byte("1.95.0\n"), 0o644))

			log := filepath.Join(t.TempDir(), "rustup.log")
			createFakeRustup(t, log)

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Env: tt.projectEnv,
			})
			err := Default.Prepare(ctx, config.Build{
				Dir:     dir,
				Targets: []string{target},
				Env:     tt.buildEnv,
			})
			require.NoError(t, err)

			got, err := os.ReadFile(log)
			require.NoError(t, err)
			gotLog := strings.ReplaceAll(string(got), "\r\n", "\n")
			wantDir := filepath.Join(folder, dir)
			wantDir, err = filepath.EvalSymlinks(wantDir)
			require.NoError(t, err)
			require.Contains(t, gotLog, "cwd="+wantDir+"\n")
			require.Contains(t, gotLog, "toolchain="+tt.wantToolchain+"\n")
			require.Contains(t, gotLog, "args=target add aarch64-unknown-linux-gnu\n")
		})
	}
}

func TestBuildWorkspaceErrorShowsAllMembers(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[workspace]
members = ["crate-a", "crate-b", "crate-c"]
`), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "dist",
		Builds: []config.Build{{
			Dir:   ".",
			Flags: []string{"--release"},
		}},
	})

	target, err := Default.Parse("aarch64-unknown-linux-gnu")
	require.NoError(t, err)

	err = Default.Build(ctx, ctx.Config.Builds[0], api.Options{
		Name:   "proj",
		Path:   "dist/proj",
		Target: target,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "crate-a")
	require.Contains(t, err.Error(), "crate-b")
	require.Contains(t, err.Error(), "crate-c")
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
			require.NoError(t, os.WriteFile("Cargo.toml", []byte(`
[package]
name = "app"
version = "0.1.0"
edition = "2021"
`), 0o644))
			createFakeCargoBuild(t, filepath.Join("target", "aarch64-apple-darwin", "release", "app"), "built by cargo")

			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				ProjectName: "app",
			})
			build, err := Default.WithDefaults(config.Build{
				ID:      "default",
				Dir:     ".",
				Targets: []string{"aarch64-apple-darwin"},
			})
			require.NoError(t, err)

			target, err := Default.Parse("aarch64-apple-darwin")
			require.NoError(t, err)
			options := api.Options{
				Name:   tt.binary,
				Path:   filepath.Join("dist", "default_aarch64-apple-darwin", tt.binary),
				Target: target,
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(options.Path), 0o755))

			require.NoError(t, Default.Build(ctx, build, options))

			got, err := os.ReadFile(options.Path)
			require.NoError(t, err)
			require.Equal(t, "built by cargo", string(got))

			bins := ctx.Artifacts.List()
			require.Len(t, bins, 1)
			require.Equal(t, tt.binary, bins[0].Name)
			require.Equal(t, filepath.ToSlash(options.Path), bins[0].Path)
			require.Equal(t, "app", bins[0].Extra[artifact.ExtraBinary])
		})
	}
}

func createFakeRustup(tb testing.TB, log string) {
	tb.Helper()
	dir := tb.TempDir()
	name := "rustup"
	script := fmt.Sprintf(`#!/bin/sh
{
	printf 'cwd=%%s\n' "$(pwd)"
	printf 'toolchain=%%s\n' "$RUSTUP_TOOLCHAIN"
	printf 'args=%%s\n' "$*"
} > %q
`, log)
	if runtime.GOOS == "windows" {
		name += ".bat"
		log = filepath.ToSlash(log)
		script = fmt.Sprintf(`@echo off
> "%s" echo cwd=%%CD%%
>> "%s" echo toolchain=%%RUSTUP_TOOLCHAIN%%
>> "%s" echo args=%%*
`, log, log, log)
	}
	require.NoError(tb, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func createFakeCargoBuild(tb testing.TB, output, contents string) {
	tb.Helper()
	dir := tb.TempDir()
	name := "cargo"
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
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")

	folder := testlib.Mktmp(t)
	// CI (mlugg/setup-zig) forces a shared zig cache at the repo root, which
	// gets corrupted when this package and the zig package build in parallel.
	// Use a per-test cache so they cannot race.
	t.Setenv("ZIG_LOCAL_CACHE_DIR", filepath.Join(folder, ".zig-cache"))
	testlib.SharedZigCache(t)
	_, err := exec.CommandContext(t.Context(), "cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	f, err := os.OpenFile("Cargo.toml", os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("\n[profile.release]\nopt-level = 0\n")
	require.NoError(t, f.Close())
	require.NoError(t, err)

	target := "aarch64-unknown-linux-gnu.2.17"
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				Targets:      []string{target},
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Flags:        []string{"--release"},
			},
		},
	})

	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Prepare(ctx, build))

	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj-"+target, "proj"),
	}
	options.Target, err = Default.Parse(target)
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
		Goos:   "linux",
		Goarch: "arm64",
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:   "proj",
			artifact.ExtraBuilder:  "rust",
			artifact.ExtraExt:      "",
			artifact.ExtraID:       "default",
			artifact.ExtranDynLink: true,
			keyAbi:                 "gnu",
			keyLibc:                "2.17",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}

func TestBuildArm(t *testing.T) {
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")

	folder := testlib.Mktmp(t)
	// CI (mlugg/setup-zig) forces a shared zig cache at the repo root, which
	// gets corrupted when this package and the zig package build in parallel.
	// Use a per-test cache so they cannot race.
	t.Setenv("ZIG_LOCAL_CACHE_DIR", filepath.Join(folder, ".zig-cache"))
	testlib.SharedZigCache(t)
	_, err := exec.CommandContext(t.Context(), "cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	f, err := os.OpenFile("Cargo.toml", os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("\n[profile.release]\nopt-level = 0\n")
	require.NoError(t, f.Close())
	require.NoError(t, err)

	target := "armv7-unknown-linux-gnueabihf.2.17"
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				Targets:      []string{target},
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Flags:        []string{"--release"},
			},
		},
	})

	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Prepare(ctx, build))

	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj-"+target, "proj"),
	}
	options.Target, err = Default.Parse(target)
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
		Goos:   "linux",
		Goarch: "arm",
		Goarm:  "7",
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:   "proj",
			artifact.ExtraBuilder:  "rust",
			artifact.ExtraExt:      "",
			artifact.ExtraID:       "default",
			artifact.ExtranDynLink: true,
			keyAbi:                 "gnueabihf",
			keyLibc:                "2.17",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}

func TestParse(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, err := Default.Parse("a-b")
		require.Error(t, err)
	})

	t.Run("triplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-apple-darwin")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-apple-darwin",
			Os:     "darwin",
			Arch:   "arm64",
			Vendor: "apple",
		}, target)
	})

	t.Run("quadruplet", func(t *testing.T) {
		target, err := Default.Parse("aarch64-pc-windows-gnullvm")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-pc-windows-gnullvm",
			Os:     "windows",
			Arch:   "arm64",
			Vendor: "pc",
			Abi:    "gnullvm",
		}, target)
	})
	t.Run("glibc-version", func(t *testing.T) {
		target, err := Default.Parse("aarch64-unknown-linux-gnu.2.17")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "aarch64-unknown-linux-gnu.2.17",
			Os:     "linux",
			Arch:   "arm64",
			Vendor: "unknown",
			Abi:    "gnu",
			Libc:   "2.17",
		}, target)
	})
	t.Run("glibc-version-gnueabihf", func(t *testing.T) {
		target, err := Default.Parse("armv7-unknown-linux-gnueabihf.2.17")
		require.NoError(t, err)
		require.Equal(t, Target{
			Target: "armv7-unknown-linux-gnueabihf.2.17",
			Os:     "linux",
			Arch:   "arm",
			Arm:    "7",
			Vendor: "unknown",
			Abi:    "gnueabihf",
			Libc:   "2.17",
		}, target)
	})
}

func TestStripGlibcVersion(t *testing.T) {
	for name, tt := range map[string]struct {
		input string
		want  string
		ok    bool
	}{
		"gnu":       {"aarch64-unknown-linux-gnu.2.17", "aarch64-unknown-linux-gnu", true},
		"gnueabihf": {"armv7-unknown-linux-gnueabihf.2.17", "armv7-unknown-linux-gnueabihf", true},
		"gnueabi":   {"arm-unknown-linux-gnueabi.2.31", "arm-unknown-linux-gnueabi", true},
		"no-suffix": {"aarch64-unknown-linux-gnu", "aarch64-unknown-linux-gnu", false},
		"musl":      {"aarch64-unknown-linux-musl.2.17", "aarch64-unknown-linux-musl.2.17", false},
		"gnullvm":   {"aarch64-pc-windows-gnullvm.2.17", "aarch64-pc-windows-gnullvm.2.17", false},
		"no-dashes": {"nodashes.2.17", "nodashes.2.17", false},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := stripGlibcVersion(tt.input)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsSettingPackage(t *testing.T) {
	for name, tt := range map[string]struct {
		flags  []string
		expect bool
	}{
		"not set":   {[]string{"--release", "--something-else", "--in-the-p=middle", "--something"}, false},
		"-p":        {[]string{"--release", "-p=foo", "--something"}, true},
		"--package": {[]string{"--release", "--package=foo", "--something"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			got := isSettingPackage(tt.flags)
			require.Equal(t, tt.expect, got)
		})
	}
}
