package gomod

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Builds: []config.Build{
				{
					Builder: "zig",
				},
			},
		})
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Builds: []config.Build{
				{
					Builder: "go",
				},
				{
					Builder: "zig",
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRun(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser/v2", ctx.ModulePath)
}

func TestRunGoWork(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "go.mod"),
		[]byte("module a"),
		0o666,
	))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "b"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "b", "go.mod"),
		[]byte("module a/b"),
		0o666,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "go.work"),
		[]byte("use (\n\t.\n\tb\n)"),
		0o666,
	))
	out, err := exec.Command("go", "list", "-m").CombinedOutput()
	require.NoError(t, err)
	require.Equal(t, "a\na/b", strings.TrimSpace(string(out)))
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "a", ctx.ModulePath)
}

func TestRunCustomMod(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			Mod: "readonly",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser/v2", ctx.ModulePath)
}

func TestCustomEnv(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "go.bin")
	content := []byte("#!/bin/sh\nenv | grep -qw FOO=bar")
	if testlib.IsWindows() {
		bin = strings.Replace(bin, ".bin", ".bat", 1)
		content = []byte("@echo off\r\nif not \"%FOO%\"==\"bar\" exit /b 1")
	}
	require.NoError(t, os.WriteFile(bin, content, 0o755))
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			GoBinary: bin,
			Env:      []string{"FOO=bar"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestRunCustomDir(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "src/main.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o666,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "src/go.mod"),
		[]byte("module foo"),
		0o666,
	))
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			Dir: filepath.Join(dir, "src"),
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "foo", ctx.ModulePath)
}

func TestRunOutsideGoModule(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "main.go"),
		[]byte("package main\nfunc main() {println(0)}"),
		0o666,
	))
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ModulePath)
}

func TestRunCommandError(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			GoBinary: "not-a-valid-binary",
		},
	})
	path := "$PATH"
	if testlib.IsWindows() {
		path = "%PATH%"
	}
	require.ErrorContains(
		t,
		Pipe{}.Run(ctx),
		`failed to get module path: exec: "not-a-valid-binary": executable file not found in `+path,
	)
	require.Empty(t, ctx.ModulePath)
}

func TestRunOldGoVersion(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "go.bin")
	content := []byte("#!/bin/sh\necho \"flag provided but not defined: -m\"\nexit 1")
	if testlib.IsWindows() {
		bin = strings.Replace(bin, ".bin", ".bat", 1)
		content = []byte("@echo off\r\necho flag provided but not defined: -m\r\nexit /b 1")
	}
	require.NoError(t, os.WriteFile(bin, content, 0o755))
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			GoBinary: bin,
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ModulePath)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
