package gomod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser", ctx.ModulePath)
}

func TestRunCustomMod(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			Mod: "readonly",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser", ctx.ModulePath)
}

func TestCustomEnv(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "go.bin")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nenv | grep -qw FOO=bar"), 0o755))
	ctx := testctx.NewWithCfg(config.Project{
		GoMod: config.GoMod{
			GoBinary: bin,
			Env:      []string{"FOO=bar"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestRunOutsideGoModule(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {println(0)}"), 0o666))
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
	require.EqualError(t, Pipe{}.Run(ctx), "failed to get module path: exec: \"not-a-valid-binary\": executable file not found in $PATH: ")
	require.Empty(t, ctx.ModulePath)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
