package upx

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		UPXs: []config.UPX{
			{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.UPXs, 1)
	require.Equal(t, "upx", ctx.Config.UPXs[0].Binary)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			UPXs: []config.UPX{},
		})

		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("do not skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			UPXs: []config.UPX{
				{},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRun(t *testing.T) {
	bin := "./testdata/fakeupx"
	if testlib.IsWindows() {
		bin += ".bat"
	}
	fakeupx, err := filepath.Abs(bin)
	require.NoError(t, err)

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		UPXs: []config.UPX{
			{
				Enabled: "true",
				IDs:     []string{"1"},
				Binary:  fakeupx,
			},
			{
				Enabled: "false",
			},
			{
				Enabled:  "true",
				IDs:      []string{"2"},
				Compress: "best",
				Binary:   fakeupx,
			},
			{
				Enabled:  "true",
				IDs:      []string{"3"},
				Compress: "9",
				Binary:   fakeupx,
			},
			{
				Enabled:  "true",
				IDs:      []string{"4"},
				Compress: "8",
				LZMA:     true,
				Binary:   fakeupx,
			},
			{
				Enabled: "false",
			},
			{
				Enabled: `{{ eq .Env.UPX "1" }}`,
				IDs:     []string{"5"},
				Brute:   true,
				Binary:  fakeupx,
			},
		},
	}, testctx.WithEnv(map[string]string{"UPX": "1"}))

	tmp := t.TempDir()

	var expect []string

	for _, goos := range []string{"linux", "windows", "darwin"} {
		for _, goarch := range []string{"386", "amd64", "arm64"} {
			ext := ""
			if goos == "windows" {
				ext = ".exe"
			}
			if goos == "darwin" && goarch == "386" {
				continue
			}
			if goos == "windows" && goarch == "arm64" {
				continue
			}
			for i := 1; i <= 5; i++ {
				path := filepath.Join(tmp, fmt.Sprintf("bin_%d_%s_%s%s", i, goos, goarch, ext))
				require.NoError(t, os.WriteFile(path, []byte("fake bin"), 0o755))
				expect = append(expect, path)
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "bin",
					Path:   path,
					Goos:   goos,
					Goarch: goarch,
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: strconv.Itoa(i),
					},
				})
			}
		}
	}

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))

	for _, path := range expect {
		require.FileExists(t, path+".ran")
	}
}

func TestEnabled(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			UPXs: []config.UPX{
				{},
			},
		})

		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("tmpl", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			UPXs: []config.UPX{
				{
					Enabled: `{{ printf "false" }}`,
				},
			},
		})

		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			UPXs: []config.UPX{
				{
					Enabled: `{{ .Foo }}`,
				},
			},
		})

		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func TestUpxNotInstalled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		UPXs: []config.UPX{
			{
				Enabled: "true",
				Binary:  "fakeupx",
			},
		},
	})

	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}
