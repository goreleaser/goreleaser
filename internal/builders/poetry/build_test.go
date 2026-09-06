package poetry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pyproject"
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
			Tool:    "poetry",
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

func TestArtifactNames(t *testing.T) {
	var proj pyproject.PyProject
	proj.Project.Name = "My..Pkg"
	proj.Project.Version = "0.1.0"

	options := api.Options{
		Path:   filepath.Join("dist", "my-pkg-all-all", "my-pkg"),
		Target: Target{},
	}
	build := config.Build{ID: "my-pkg"}

	testlib.RequireEqualArtifacts(t, []*artifact.Artifact{
		{
			Name:   "my_pkg-0.1.0-py3-none-any.whl",
			Path:   filepath.Join("dist", "my-pkg-all-all", "my_pkg-0.1.0-py3-none-any.whl"),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PyWheel,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "poetry",
				artifact.ExtraExt:     ".whl",
				artifact.ExtraID:      "my-pkg",
			},
		},
		{
			Name:   "my_pkg-0.1.0.tar.gz",
			Path:   filepath.Join("dist", "my-pkg-all-all", "my_pkg-0.1.0.tar.gz"),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PySdist,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "poetry",
				artifact.ExtraExt:     ".tar.gz",
				artifact.ExtraID:      "my-pkg",
			},
		},
	}, []*artifact.Artifact{
		wheel(proj, build, options),
		sdist(proj, build, options),
	})
}

func TestBuildLegacyPoetryProject(t *testing.T) {
	folder := testlib.Mktmp(t)
	createFakePoetry(t)

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        filepath.Join(folder, "dist"),
		ProjectName: "testdata",
		Builds: []config.Build{
			{
				ID:           "testdata-wheel",
				Dir:          poetryTestdataDir(t),
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Buildmode:    "wheel",
			},
			{
				ID:           "testdata-sdist",
				Dir:          poetryTestdataDir(t),
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Buildmode:    "sdist",
			},
		},
	})

	dir := filepath.Join(folder, "dist", "testdata-all-all", "testdata")
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

	builds := ctx.Artifacts.List()
	require.Len(t, builds, 2)
	testlib.RequireEqualArtifacts(t, []*artifact.Artifact{
		{
			Name:   "testdata-0.1.0-py3-none-any.whl",
			Path:   filepath.ToSlash(filepath.Join("dist", "testdata-all-all", "testdata-0.1.0-py3-none-any.whl")),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PyWheel,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "poetry",
				artifact.ExtraExt:     ".whl",
				artifact.ExtraID:      "testdata-wheel",
			},
		},
		{
			Name:   "testdata-0.1.0.tar.gz",
			Path:   filepath.ToSlash(filepath.Join("dist", "testdata-all-all", "testdata-0.1.0.tar.gz")),
			Goos:   "all",
			Goarch: "all",
			Target: "none-any",
			Type:   artifact.PySdist,
			Extra: artifact.Extras{
				artifact.ExtraBuilder: "poetry",
				artifact.ExtraExt:     ".tar.gz",
				artifact.ExtraID:      "testdata-sdist",
			},
		},
	}, builds)

	for _, art := range builds {
		_, err := art.Checksum("sha256")
		require.NoError(t, err)
		fi, err := os.Stat(art.Path)
		require.NoError(t, err)
		require.True(t, modTime.Equal(fi.ModTime()))
	}
}

func TestBuild(t *testing.T) {
	testlib.CheckPath(t, "poetry")

	folder := testlib.Mktmp(t)
	cmd := exec.CommandContext(t.Context(), "poetry", "new", "proj")
	cmd.Dir = folder
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)

	t.Chdir(filepath.Join(folder, "proj"))

	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        filepath.Join(folder, "dist"),
		ProjectName: "proj",
		Builds: []config.Build{
			{
				ID:           "proj-wheel",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Buildmode:    "wheel",
			},
			{
				ID:           "proj-sdist",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Buildmode:    "sdist",
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
				artifact.ExtraBuilder: "poetry",
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
				artifact.ExtraBuilder: "poetry",
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

func createFakePoetry(tb testing.TB) {
	tb.Helper()

	createFakeExecutable(
		tb,
		"poetry",
		`#!/bin/sh
output=""
format=""
while [ "$#" -gt 0 ]; do
	case "$1" in
		--output)
			shift
			output="$1"
			;;
		--format)
			shift
			format="$1"
			;;
	esac
	shift
done
mkdir -p "$output"
case "$format" in
	wheel)
		printf fake > "$output/testdata-0.1.0-py3-none-any.whl"
		;;
	sdist)
		printf fake > "$output/testdata-0.1.0.tar.gz"
		;;
esac
`,
		`@echo off
set output=
set format=
:parse
if "%1"=="" goto done
if "%1"=="--output" (
	set "output=%~2"
	shift
	shift
	goto parse
) else if "%1"=="--format" (
	set "format=%~2"
	shift
	shift
	goto parse
)
shift
goto parse
:done
if not exist "%output%" mkdir "%output%"
if "%format%"=="wheel" (
	echo fake>"%output%\testdata-0.1.0-py3-none-any.whl"
) else if "%format%"=="sdist" (
	echo fake>"%output%\testdata-0.1.0.tar.gz"
)
`,
	)
}

func poetryTestdataDir(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(tb, ok)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func createFakeExecutable(tb testing.TB, name, unix, windows string) {
	tb.Helper()

	dir := tb.TempDir()
	if runtime.GOOS == "windows" {
		name += ".bat"
		unix = windows
	}
	require.NoError(tb, os.WriteFile(filepath.Join(dir, name), []byte(unix), 0o755))
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
