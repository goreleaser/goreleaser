package uv

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
	t.Run("valid", func(t *testing.T) {
		got, err := Default.Parse(defaultTarget)
		require.NoError(t, err)
		require.IsType(t, Target{}, got)
	})
	t.Run("invalid", func(t *testing.T) {
		got, err := Default.Parse(defaultTarget)
		require.NoError(t, err)
		require.IsType(t, Target{}, got)
	})
}

func TestWithDefaults(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		build, err := Default.WithDefaults(config.Build{
			Dir: "./testdata",
		})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "uv",
			Command: "build",
			Dir:     "./testdata",
			Binary:  "testdata-0.1.0-py3-none-any",
			Targets: []string{defaultTarget},
			BuildDetails: config.BuildDetails{
				Buildmode: "wheel",
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
	testlib.CheckPath(t, "uv")

	folder := testlib.Mktmp(t)
	cmd := exec.Command("uv", "init", "--name", "proj")
	cmd.Dir = folder
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        filepath.Join(folder, "dist"),
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "wheel",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Buildmode: "wheel",
				},
			},
			{
				ID:           "sdist",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Buildmode: "sdist",
				},
			},
		},
	})

	wheelBuild, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	wheelOptions := api.Options{
		Name:   wheelBuild.Binary + ".whl",
		Path:   filepath.Join("dist", "proj-all-all", wheelBuild.Binary+".whl"),
		Ext:    ".whl",
		Target: Target{},
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(wheelOptions.Path), 0o755)) // this happens on internal/pipe/build/ when in prod
	require.NoError(t, Default.Build(ctx, wheelBuild, wheelOptions))

	sdistBuild, err := Default.WithDefaults(ctx.Config.Builds[1])
	require.NoError(t, err)
	sdistOptions := api.Options{
		Name:   sdistBuild.Binary + ".tar.gz",
		Path:   filepath.Join("dist", "proj-all-all", sdistBuild.Binary+".tar.gz"),
		Ext:    ".tar.gz",
		Target: Target{},
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(sdistOptions.Path), 0o755)) // this happens on internal/pipe/build/ when in prod
	require.NoError(t, Default.Build(ctx, sdistBuild, sdistOptions))

	list := ctx.Artifacts
	require.NoError(t, list.Visit(func(a *artifact.Artifact) error {
		s, err := filepath.Rel(folder, a.Path)
		if err == nil {
			a.Path = s
		}
		return nil
	}))

	builds := list.List()
	require.Len(t, builds, 2)

	testlib.RequireEqualArtifacts(t, []*artifact.Artifact{
		{
			Name:   "proj-0.1.0-py3-none-any.whl",
			Path:   filepath.ToSlash(wheelOptions.Path),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PyWheel,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "uv",
				artifact.ExtraExt:     ".whl",
				artifact.ExtraID:      "wheel",
			},
		},
		{
			Name:   "proj-0.1.0.tar.gz",
			Path:   filepath.ToSlash(sdistOptions.Path),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PySdist,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "uv",
				artifact.ExtraExt:     ".tar.gz",
				artifact.ExtraID:      "sdist",
			},
		},
	}, builds)

	for _, art := range builds {
		require.FileExists(t, art.Path)
		fi, err := os.Stat(art.Path)
		require.NoError(t, err)
		require.True(t, modTime.Equal(fi.ModTime()))
	}
}
