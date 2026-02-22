//go:build integration

package rust

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
	testlib.CheckPath(t, "cargo")
	testlib.CheckPath(t, "cargo-zigbuild")

	folder := testlib.Mktmp(t)
	_, err := exec.CommandContext(t.Context(), "cargo", "init", "--bin", "--name=proj").CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        "dist",
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "default",
				Dir:          ".",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Flags: []string{"--release"},
				},
			},
		},
	})

	build, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	require.NoError(t, Default.Prepare(ctx, build))

	target := "aarch64-unknown-linux-gnu"
	options := api.Options{
		Name: "proj",
		Path: filepath.Join("dist", "proj-"+target, "proj"),
	}
	options.Target, err = Default.Parse(target)
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
		Goos:   "linux",
		Goarch: "arm64",
		Target: target,
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraBinary:   "proj",
			artifact.ExtraBuilder:  "rust",
			artifact.ExtraExt:      "",
			artifact.ExtraID:       "default",
			artifact.ExtranDynLink: true,
			keyAbi:                 "gnu",
		},
	}, *bin)

	require.FileExists(t, bin.Path)
	fi, err := os.Stat(bin.Path)
	require.NoError(t, err)
	require.True(t, modTime.Equal(fi.ModTime()))
}
