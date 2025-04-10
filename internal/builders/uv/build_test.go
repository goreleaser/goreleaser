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
			InternalDefaults: config.BuildInternalDefaults{
				Binary: true,
			},
		})
		require.NoError(t, err)
		require.Equal(t, config.Build{
			Tool:    "uv",
			Command: "build",
			Dir:     "./testdata",
			Targets: []string{defaultTarget},
			InternalDefaults: config.BuildInternalDefaults{
				Binary: true,
			},
		}, build)
	})

	t.Run("user set binary", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:    "./testdata",
			Binary: "a",
		})
		require.ErrorIs(t, err, errSetBinary)
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:     "./testdata",
			Targets: []string{"a-b"},
		})
		require.ErrorIs(t, err, errTargets)
	})

	t.Run("invalid config option", func(t *testing.T) {
		_, err := Default.WithDefaults(config.Build{
			Dir:  "./testdata",
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
				ID:           "proj-wheel",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Buildmode: "wheel",
				},
			},
			{
				ID:           "proj-sdist",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				BuildDetails: config.BuildDetails{
					Buildmode: "sdist",
				},
			},
		},
	})

	dir := filepath.Join("dist", "proj-all-all", "proj")
	require.NoError(t, os.MkdirAll(filepath.Dir(dir), 0o755)) // this happens on internal/pipe/build/ when in prod
	for _, build := range ctx.Config.Builds {
		build, err := Default.WithDefaults(build)
		require.NoError(t, err)
		opts := api.Options{
			Path:   dir,
			Target: Target{},
		}
		require.NoError(t, Default.Build(ctx, build, opts))
	}

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
			Path:   "dist/proj-all-all/proj-0.1.0-py3-none-any.whl",
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PyWheel,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "uv",
				artifact.ExtraExt:     ".whl",
				artifact.ExtraID:      "proj-wheel",
			},
		},
		{
			Name:   "proj-0.1.0.tar.gz",
			Path:   "dist/proj-all-all/proj-0.1.0.tar.gz",
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PySdist,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "uv",
				artifact.ExtraExt:     ".tar.gz",
				artifact.ExtraID:      "proj-sdist",
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

func TestBuildSpecificModes(t *testing.T) {
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

	dir := filepath.Join("dist", "proj-all-all")
	require.NoError(t, os.MkdirAll(dir, 0o755)) // this happens on internal/pipe/build/ when in prod

	wheelBuild, err := Default.WithDefaults(ctx.Config.Builds[0])
	require.NoError(t, err)
	wheelOptions := api.Options{
		Path:   filepath.Join(dir, "proj"),
		Target: Target{},
	}
	require.NoError(t, Default.Build(ctx, wheelBuild, wheelOptions))

	sdistBuild, err := Default.WithDefaults(ctx.Config.Builds[1])
	require.NoError(t, err)
	sdistOptions := api.Options{
		Path:   filepath.Join(dir, "proj"),
		Target: Target{},
	}
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
			Path:   "dist/proj-all-all/proj-0.1.0-py3-none-any.whl",
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
			Path:   "dist/proj-all-all/proj-0.1.0.tar.gz",
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
