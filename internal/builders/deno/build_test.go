package deno

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
		"x86_64-pc-windows-msvc": {
			Target: "x86_64-pc-windows-msvc",
			Os:     "windows",
			Arch:   "x86_64",
			Abi:    "msvc",
			Vendor: "pc",
		},
		"aarch64-apple-darwin": {
			Target: "aarch64-apple-darwin",
			Os:     "darwin",
			Arch:   "aarch64",
			Vendor: "apple",
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
			Tool:    "deno",
			Command: "compile",
			Dir:     ".",
			Main:    "main.ts",
			Targets: defaultTargets(),
		}, build)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Targets: []string{"a-b"},
		})
		require.Error(t, err)
	})
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "deno")
	folder := testlib.Mktmp(t)
	_, err := exec.Command("deno", "init").CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
			},
		},
	})
	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)

	options := api.Options{
		Name:   "proj",
		Path:   filepath.Join("dist", "proj-darwin-arm64", "proj"),
		Target: nil,
	}
	options.Target, err = Default.Parse("aarch64-apple-darwin")
	require.NoError(t, err)

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
		Target: "aarch64-apple-darwin",
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:  "proj",
			artifact.ExtraBuilder: "deno",
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

func TestIsValid(t *testing.T) {
	for _, target := range []string{
		"x86_64-pc-windows-msvc",
		"x86_64-apple-darwin",
		"aarch64-apple-darwin",
		"x86_64-unknown-linux-gnu",
		"aarch64-unknown-linux-gnu",
	} {
		t.Run(target, func(t *testing.T) {
			require.True(t, isValid(target))
		})
	}

	t.Run("invalid", func(t *testing.T) {
		require.False(t, isValid("foo-bar"))
	})
}
