package upx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
		ctx := testctx.NewWithCfg(config.Project{
			UPXs: []config.UPX{},
		})
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("do not skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			UPXs: []config.UPX{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRun(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		UPXs: []config.UPX{
			{
				Enabled: true,
				IDs:     []string{"1"},
			},
			{
				Enabled:  true,
				IDs:      []string{"2"},
				Compress: "best",
			},
			{
				Enabled:  true,
				IDs:      []string{"3"},
				Compress: "9",
			},
			{
				Enabled:  true,
				IDs:      []string{"4"},
				Compress: "8",
				LZMA:     true,
			},
			{
				Enabled: true,
				IDs:     []string{"5"},
				Brute:   true,
			},
		},
	})

	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.go")
	require.NoError(t, os.WriteFile(main, []byte("package main\nfunc main(){ println(1) }"), 0o644))

	for _, goos := range []string{"linux", "windows", "darwin"} {
		for _, goarch := range []string{"386", "amd64", "arm64"} {
			ext := ""
			if goos == "windows" {
				ext = ".exe"
			}
			path := filepath.Join(tmp, fmt.Sprintf("bin_%s_%s%s", goos, goarch, ext))
			cmd := exec.Command("go", "build", "-o", path, main)
			cmd.Env = append([]string{
				"CGO_ENABLED=0",
				"GOOS=" + goos,
				"GOARCH=" + goarch,
			}, cmd.Environ()...)
			if cmd.Run() != nil {
				// ignore unsupported arches
				continue
			}

			for i := 1; i <= 5; i++ {
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "bin",
					Path:   path,
					Goos:   goos,
					Goarch: goarch,
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: fmt.Sprintf("%d", i),
					},
				})
			}

		}
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
}

func TestDisabled(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		UPXs: []config.UPX{
			{},
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestUpxNotInstalled(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		UPXs: []config.UPX{
			{
				Enabled: true,
				Binary:  "fakeupx",
			},
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}
