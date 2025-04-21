package zig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
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
			BuildDetails: config.BuildDetails{
				Flags: []string{"-Doptimize=ReleaseSafe"},
			},
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Main: "something",
		})
		require.Error(t, err)
	})
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "zig")

	folder := testlib.Mktmp(t)
	folder = filepath.Join(folder, "proj")
	require.NoError(t, os.MkdirAll(folder, 0o755))
	cmd := exec.Command("zig", "init")
	cmd.Dir = folder
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Env: []string{
			"OPTIMIZE_FOR=ReleaseSmall",
		},
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          "./proj/",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Flags: []string{"-Doptimize={{.Env.OPTIM}}"},
					Env: []string{
						"OPTIM={{.Env.OPTIMIZE_FOR}}",
					},
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
