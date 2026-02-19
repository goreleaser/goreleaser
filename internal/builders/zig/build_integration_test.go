//go:build integration

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

func TestIntegrationBuild(t *testing.T) {
	testlib.CheckPath(t, "zig")

	folder := testlib.Mktmp(t)
	folder = filepath.Join(folder, "proj")
	require.NoError(t, os.MkdirAll(folder, 0o755))
	cmd := exec.CommandContext(t.Context(), "zig", "init")
	cmd.Dir = folder
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
