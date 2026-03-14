package flatpak

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Flatpaks: []config.Flatpak{{}},
		}, testctx.Skip(skips.Flatpak))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Flatpaks: []config.Flatpak{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{validFlatpak()},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Flatpaks[0].NameTemplate)
}

func TestDefaultMissingFields(t *testing.T) {
	for name, mod := range map[string]func(*config.Flatpak){
		"no app_id":          func(fp *config.Flatpak) { fp.AppID = "" },
		"no runtime":         func(fp *config.Flatpak) { fp.Runtime = "" },
		"no runtime_version": func(fp *config.Flatpak) { fp.RuntimeVersion = "" },
		"no sdk":             func(fp *config.Flatpak) { fp.SDK = "" },
	} {
		t.Run(name, func(t *testing.T) {
			fp := validFlatpak()
			mod(&fp)
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Flatpaks: []config.Flatpak{fp},
			})
			require.Error(t, Pipe{}.Default(ctx))
		})
	}
}

func TestSeveralFlatpaksWithTheSameID(t *testing.T) {
	fp1 := validFlatpak()
	fp1.ID = "a"
	fp2 := validFlatpak()
	fp2.ID = "a"
	fp2.AppID = "org.example.App2"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp1, fp2},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 flatpaks with the ID 'a', please fix your config")
}

func TestNoFlatpakBuilderInPath(t *testing.T) {
	t.Setenv("PATH", "")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{validFlatpak()},
	})
	require.ErrorIs(t, Pipe{}.Run(ctx), ErrNoFlatpakBuilder)
}

func TestRunPipeDisabled(t *testing.T) {
	fp := validFlatpak()
	fp.Disable = "true"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeDisabledTemplate(t *testing.T) {
	fp := validFlatpak()
	fp.Disable = "{{.Env.SKIP}}"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{fp},
	}, testctx.WithEnv(map[string]string{"SKIP": "true"}))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	testlib.OnlyOnLinux(t, "flatpak only works on linux")
	testlib.CheckPath(t, "flatpak-builder")
	dist := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	fp := validFlatpak()
	fp.NameTemplate = "foo_{{.Arch}"
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Dist:        dist,
		Flatpaks:    []config.Flatpak{fp},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestRunPipe(t *testing.T) {
	testlib.OnlyOnLinux(t, "flatpak only works on linux")
	testlib.CheckPath(t, "flatpak-builder")
	testlib.CheckPath(t, "flatpak")
	dist := filepath.Join(t.TempDir(), "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	fp := validFlatpak()
	fp.NameTemplate = "foo_{{.Arch}}"
	fp.AppID = "org.example.MyBin"
	fp.IDs = []string{"foo"}
	fp.Command = "foo"
	fp.FinishArgs = []string{"--share=network", "--socket=x11"}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Flatpaks:    []config.Flatpak{fp},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	requireNoGerror(t, Pipe{}.Run(ctx))

	list := ctx.Artifacts.Filter(artifact.ByType(artifact.Flatpak)).List()
	require.NotEmpty(t, list)

	manifestFile := filepath.Join(dist, "flatpak", "foo_amd64", "x86_64", "org.example.MyBin.json")
	manifestBytes, err := os.ReadFile(manifestFile)
	require.NoError(t, err)

	var manifest Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.Equal(t, "org.example.MyBin", manifest.ID)
	require.Equal(t, "org.freedesktop.Platform", manifest.Runtime)
	require.Equal(t, "24.08", manifest.RuntimeVersion)
	require.Equal(t, "org.freedesktop.Sdk", manifest.SDK)
	require.Equal(t, "foo", manifest.Command)
	require.Equal(t, []string{"--share=network", "--socket=x11"}, manifest.FinishArgs)
	require.Len(t, manifest.Modules, 1)
	require.Equal(t, "simple", manifest.Modules[0].BuildSystem)
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"flatpak-builder", "flatpak"}, Pipe{}.Dependencies(nil))
}

func validFlatpak() config.Flatpak {
	return config.Flatpak{
		AppID:          "org.example.App",
		Runtime:        "org.freedesktop.Platform",
		RuntimeVersion: "24.08",
		SDK:            "org.freedesktop.Sdk",
	}
}

func requireNoGerror(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		return
	}
	gerr, ok := errors.AsType[gerrors.ErrDetailed](err)
	require.True(tb, ok)
	require.NoError(tb, err, "messages: %v, details: %v, output: %s", gerr.Messages(), maps.Collect(gerr.Details()), gerr.Output())
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	t.Helper()
	binPath := filepath.Join(dist, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0o755))
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			a := &artifact.Artifact{
				Name:   name,
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraID: name,
				},
			}
			if goarch == "amd64" {
				a.Goamd64 = "v1"
			}
			ctx.Artifacts.Add(a)
		}
	}
}
