package makeself

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.Equal(t, "makeself packages", Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context(), testctx.Skip(skips.Makeself))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Makeselfs: []config.Makeself{{}},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})

	t.Run("skip no makeselfs", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.True(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Makeselfs: []config.Makeself{
			{},
			{
				ID:       "custom",
				Name:     "custom",
				Filename: "custom_{{.Os}}_{{.Arch}}.bin",
				Goos:     []string{"freebsd"},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Makeselfs, 2)

	m1 := ctx.Config.Makeselfs[0]
	require.Equal(t, "default", m1.ID)
	require.NotEmpty(t, m1.Name)
	require.Equal(t, defaultNameTemplate, m1.Filename)
	require.Len(t, m1.Goos, 2)

	m2 := ctx.Config.Makeselfs[1]
	require.Equal(t, "custom", m2.ID)
	require.Equal(t, "custom", m2.Name)
	require.Equal(t, "custom_{{.Os}}_{{.Arch}}.bin", m2.Filename)
	require.Len(t, m2.Goos, 1)
}

func makeContext(tb testing.TB) *context.Context {
	tb.Helper()
	ctx := testctx.WrapWithCfg(tb.Context(), config.Project{
		ProjectName: "myproj",
		Dist:        tb.TempDir(),
	}, testctx.WithVersion("1.2.3"))

	tmp := tb.TempDir()
	require.NoError(tb, os.WriteFile(
		filepath.Join(tmp, "mybin"),
		[]byte("#!/bin/sh\necho 'hello world, from the binary'"),
		0o755,
	))
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "dir/mybin",
				Path:   filepath.Join(tmp, "mybin"),
				Type:   artifact.Binary,
				Goos:   goos,
				Goarch: goarch,
			})
		}
	}
	return ctx
}

func TestRun(t *testing.T) {
	testlib.SkipIfWindows(t, "no makeself on windows")
	testlib.CheckPath(t, "makeself")

	t.Run("simple", func(t *testing.T) {
		ctx := makeContext(t)
		ctx.Config.Makeselfs = append(ctx.Config.Makeselfs, config.Makeself{
			ID:     "simple",
			Script: "./testdata/setup.sh",
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		result := ctx.Artifacts.Filter(
			artifact.And(
				artifact.ByType(artifact.Makeself),
				artifact.ByID("simple"),
			)).List()
		require.Len(t, result, 4)

		for _, m := range result {
			require.Equal(t, "simple", artifact.ExtraOr(*m, artifact.ExtraID, ""))
			require.Equal(t, "makeself", artifact.ExtraOr(*m, artifact.ExtraFormat, ""))
			require.Equal(t, ".run", artifact.ExtraOr(*m, artifact.ExtraExt, ""))
		}

		slices.SortFunc(result, func(a, b *artifact.Artifact) int {
			return strings.Compare(a.Path, b.Path)
		})

		requireContainsFiles(t, result[0].Path, "dir/mybin", "package.lsm", "setup.sh")
		requireEqualLSM(t, result[0].Path)
		requireRunMakeself(t, result[0].Path)
	})
	t.Run("complete", func(t *testing.T) {
		ctx := makeContext(t)
		ctx.Config.Makeselfs = append(ctx.Config.Makeselfs, config.Makeself{
			ID:          "complete",
			Script:      "./testdata/setup.sh",
			Description: "My thing",
			Keywords:    []string{"one", "two"},
			Homepage:    "https://goreleaser.com",
			Maintainer:  "me",
			License:     "MIT",
			Goos:        []string{"linux"},
			Goarch:      []string{"arm64"},
			Compression: "gzip",
			ExtraArgs:   []string{"--notemp"},
			Files: []config.MakeselfFile{
				{
					Source:      "./testdata/foo.txt",
					Destination: "docs/foo.txt",
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		result := ctx.Artifacts.Filter(
			artifact.And(
				artifact.ByType(artifact.Makeself),
				artifact.ByID("complete"),
			)).List()
		require.Len(t, result, 1)

		m := result[0]
		require.Equal(t, "complete", artifact.ExtraOr(*m, artifact.ExtraID, ""))
		require.Equal(t, "makeself", artifact.ExtraOr(*m, artifact.ExtraFormat, ""))
		require.Equal(t, ".run", artifact.ExtraOr(*m, artifact.ExtraExt, ""))

		requireContainsFiles(t, result[0].Path, "dir/mybin", "package.lsm", "setup.sh", "docs/foo.txt")
		requireEqualLSM(t, result[0].Path)
		requireRunMakeself(t, result[0].Path)
	})
}

func requireEqualLSM(tb testing.TB, path string) {
	tb.Helper()
	out, err := exec.CommandContext(tb.Context(), path, "--lsm").CombinedOutput()
	require.NoError(tb, err, string(out))
	golden.RequireEqualExt(tb, out, ".lsm")
}

func requireContainsFiles(tb testing.TB, path string, files ...string) {
	tb.Helper()
	out, err := exec.CommandContext(tb.Context(), path, "--list").CombinedOutput()
	require.NoError(tb, err, string(out))
	for _, f := range files {
		require.Contains(tb, string(out), f)
	}
}

func requireRunMakeself(tb testing.TB, path string) {
	tb.Helper()
	cmd := exec.CommandContext(tb.Context(), path)
	cmd.Dir = tb.TempDir()
	out, err := cmd.CombinedOutput()
	require.NoError(tb, err, string(out))
	require.Contains(tb, string(out), "hello world, from the binary")
}
