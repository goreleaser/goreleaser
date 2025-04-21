package upx

import (
	"fmt"
	"os"
	"os/exec"
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
	bin := "./testdata/fakeupx"
	if testlib.IsWindows() {
		bin += ".bat"
	}
	fakeupx, err := filepath.Abs(bin)
	require.NoError(t, err)

	ctx := testctx.NewWithCfg(config.Project{
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
		ctx := testctx.NewWithCfg(config.Project{
			UPXs: []config.UPX{
				{},
			},
		})
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("tmpl", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			UPXs: []config.UPX{
				{
					Enabled: `{{ printf "false" }}`,
				},
			},
		})
		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})
	t.Run("invalid template", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
		UPXs: []config.UPX{
			{
				Enabled: "true",
				Binary:  "fakeupx",
			},
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestFindBinaries(t *testing.T) {
	ctx := testctx.New()
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main.go")
	require.NoError(t, os.WriteFile(main, []byte("package main\nfunc main(){ println(1) }"), 0o644))
	for _, goos := range []string{"linux", "windows", "darwin"} {
		for _, goarch := range []string{"386", "amd64", "arm64", "arm", "mips"} {
			ext := ""
			goarm := ""
			gomips := ""
			goamd64 := ""
			if goos == "windows" {
				ext = ".exe"
			}
			switch goarch {
			case "arm":
				goarm = "7"
				if goos != "linux" {
					continue
				}
			case "mips":
				gomips = "softfloat"
				if goos != "linux" {
					continue
				}
			case "arm64":
				if goos == "windows" {
					continue
				}
			case "amd64":
				goamd64 = "v1"
			case "386":
				if goos == "darwin" {
					continue
				}
			}
			path := filepath.Join(tmp, fmt.Sprintf("bin_%s_%s%s", goos, goarch, ext))
			cmd := exec.Command("go", "build", "-o", path, main)
			cmd.Env = append([]string{
				"CGO_ENABLED=0",
				"GOOS=" + goos,
				"GOARCH=" + goarch,
				"GOAMD64=" + goamd64,
				"GOARM=" + goarm,
				"GOMIPS=" + gomips,
			}, cmd.Environ()...)
			if cmd.Run() != nil {
				// ignore unsupported arches
				continue
			}

			for i := 1; i <= 5; i++ {
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    "bin",
					Path:    path,
					Goos:    goos,
					Goarch:  goarch,
					Goarm:   goarm,
					Gomips:  gomips,
					Goamd64: goamd64,
					Type:    artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: fmt.Sprintf("%d", i),
					},
				})
			}
		}
	}

	t.Run("only ids", func(t *testing.T) {
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs: []string{"1", "2", "3"},
		}), 27)
	})

	t.Run("id and goos", func(t *testing.T) {
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs:  []string{"4"},
			Goos: []string{"windows", "darwin"},
		}), 4) // amd64, 386
	})

	t.Run("id, goos goarch", func(t *testing.T) {
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs:    []string{"3"},
			Goos:   []string{"windows"},
			Goarch: []string{"386", "amd64"},
		}), 2)
	})

	t.Run("goamd64", func(t *testing.T) {
		require.Empty(t, findBinaries(ctx, config.UPX{
			IDs:     []string{"2"},
			Goos:    []string{"linux"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v3"},
		}))
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs:     []string{"2"},
			Goos:    []string{"linux"},
			Goarch:  []string{"amd64"},
			Goamd64: []string{"v1", "v2", "v3", "v4"},
		}), 1)
	})

	t.Run("goarm", func(t *testing.T) {
		require.Empty(t, findBinaries(ctx, config.UPX{
			IDs:    []string{"2"},
			Goos:   []string{"linux"},
			Goarch: []string{"arm"},
			Goarm:  []string{"6"},
		}))
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs:    []string{"2"},
			Goos:   []string{"linux"},
			Goarch: []string{"arm"},
			Goarm:  []string{"7"},
		}), 1)
		require.Len(t, findBinaries(ctx, config.UPX{
			IDs:   []string{"2"},
			Goarm: []string{"6", "7"},
		}), 1)
	})
}
