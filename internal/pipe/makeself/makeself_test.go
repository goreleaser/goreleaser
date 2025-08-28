package makeself

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.Equal(t, "makeself packages", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Makeself))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Makeselfs: []config.Makeself{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no makeselfs", func(t *testing.T) {
		ctx := testctx.New()
		require.True(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Makeselfs: []config.Makeself{
			{},
			{
				ID:       "custom",
				Name:     "custom",
				Filename: "custom_{{.Os}}_{{.Arch}}.bin",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Makeselfs, 2)

	m1 := ctx.Config.Makeselfs[0]
	require.Equal(t, "default", m1.ID)
	require.NotEmpty(t, m1.Name)
	require.Equal(t, defaultNameTemplate, m1.Filename)

	m2 := ctx.Config.Makeselfs[1]
	require.Equal(t, "custom", m2.ID)
	require.Equal(t, "custom", m2.Name)
	require.Equal(t, "custom_{{.Os}}_{{.Arch}}.bin", m2.Filename)
}

func TestRunSimple(t *testing.T) {
	testlib.CheckPath(t, "makeself")
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "myproj",
		Dist:        t.TempDir(),
		Makeselfs: []config.Makeself{{
			Script: "./testdata/setup.sh",
		}},
	})
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "mybin"),
		[]byte("#!/bin/sh\necho hi"),
		0o755,
	))
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   filepath.Join(tmp, "mybin"),
				Type:   artifact.Binary,
				Goos:   goos,
				Goarch: goarch,
			})
		}
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	result := ctx.Artifacts.Filter(artifact.ByType(artifact.Makeself)).List()
	require.Len(t, result, 4)

	for _, m := range result {
		require.Equal(t, "default", artifact.ExtraOr(*m, artifact.ExtraID, ""))
		require.Equal(t, "makeself", artifact.ExtraOr(*m, artifact.ExtraFormat, ""))
		require.Equal(t, ".run", artifact.ExtraOr(*m, artifact.ExtraExt, ""))
	}

	{
		out, err := exec.CommandContext(t.Context(), result[0].Path, "--tar", "tvzf").CombinedOutput()
		require.NoError(t, err, string(out))
		require.Contains(t, string(out), "mybin")
		require.Contains(t, string(out), "package.lsm")
		require.Contains(t, string(out), "script.sh")
	}

	{
		out, err := exec.CommandContext(t.Context(), result[0].Path, "--tar", "xfO", ".package.lsm").CombinedOutput()
		require.NoError(t, err, string(out))
		golden.RequireEqualExt(t, out, ".lsm")
	}
}
