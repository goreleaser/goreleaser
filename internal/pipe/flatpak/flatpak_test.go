package flatpak

import (
	"encoding/json"
	"errors"
	"fmt"
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
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
		}},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Flatpaks[0].NameTemplate)
}

func TestDefaultNoAppID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
		}},
	})
	require.ErrorIs(t, Pipe{}.Default(ctx), ErrNoAppID)
}

func TestDefaultNoRuntime(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
		}},
	})
	require.ErrorIs(t, Pipe{}.Default(ctx), ErrNoRuntime)
}

func TestDefaultNoRuntimeVersion(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:   "org.example.App",
			Runtime: "org.freedesktop.Platform",
			SDK:     "org.freedesktop.Sdk",
		}},
	})
	require.ErrorIs(t, Pipe{}.Default(ctx), ErrNoRuntimeVersion)
}

func TestDefaultNoSDK(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
		}},
	})
	require.ErrorIs(t, Pipe{}.Default(ctx), ErrNoSDK)
}

func TestSeveralFlatpaksWithTheSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{
			{
				ID:             "a",
				AppID:          "org.example.App",
				Runtime:        "org.freedesktop.Platform",
				RuntimeVersion: "24.08",
				SDK:            "org.freedesktop.Sdk",
			},
			{
				ID:             "a",
				AppID:          "org.example.App2",
				Runtime:        "org.freedesktop.Platform",
				RuntimeVersion: "24.08",
				SDK:            "org.freedesktop.Sdk",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 flatpaks with the ID 'a', please fix your config")
}

func TestNoFlatpakBuilderInPath(t *testing.T) {
	t.Setenv("PATH", "")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
		}},
	})
	require.ErrorIs(t, Pipe{}.Run(ctx), ErrNoFlatpakBuilder)
}

func TestRunPipeDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
			Disable:        "true",
		}},
	})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeDisabledTemplate(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Flatpaks: []config.Flatpak{{
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
			Disable:        "{{.Env.SKIP}}",
		}},
	}, testctx.WithEnv(map[string]string{"SKIP": "true"}))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	testlib.CheckPath(t, "flatpak-builder")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Dist:        dist,
		Flatpaks: []config.Flatpak{{
			NameTemplate:   "foo_{{.Arch}",
			AppID:          "org.example.App",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
		}},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestRunPipe(t *testing.T) {
	testlib.CheckPath(t, "flatpak-builder")
	testlib.CheckPath(t, "flatpak")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Flatpaks: []config.Flatpak{{
			NameTemplate:   "foo_{{.Arch}}",
			AppID:          "org.example.MyBin",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
			IDs:            []string{"foo"},
			FinishArgs:     []string{"--share=network"},
		}},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	requireNoGerror(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.Flatpak)).List()
	require.NotEmpty(t, list)
}

func TestRunPipeManifest(t *testing.T) {
	testlib.CheckPath(t, "flatpak-builder")
	testlib.CheckPath(t, "flatpak")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "testproject",
		Dist:        dist,
		Flatpaks: []config.Flatpak{{
			NameTemplate:   "foo_{{.Arch}}",
			AppID:          "org.example.TestProject",
			Runtime:        "org.freedesktop.Platform",
			RuntimeVersion: "24.08",
			SDK:            "org.freedesktop.Sdk",
			Command:        "mycommand",
			FinishArgs:     []string{"--share=network", "--socket=x11"},
		}},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)

	requireNoGerror(t, Pipe{}.Run(ctx))

	manifestFile := filepath.Join(dist, "flatpak", "foo_amd64", "org.example.TestProject.json")
	manifestBytes, err := os.ReadFile(manifestFile)
	require.NoError(t, err)

	var manifest Manifest
	require.NoError(t, json.Unmarshal(manifestBytes, &manifest))
	require.Equal(t, "org.example.TestProject", manifest.ID)
	require.Equal(t, "org.freedesktop.Platform", manifest.Runtime)
	require.Equal(t, "24.08", manifest.RuntimeVersion)
	require.Equal(t, "org.freedesktop.Sdk", manifest.SDK)
	require.Equal(t, "mycommand", manifest.Command)
	require.Equal(t, []string{"--share=network", "--socket=x11"}, manifest.FinishArgs)
	require.Len(t, manifest.Modules, 1)
	require.Equal(t, "simple", manifest.Modules[0].BuildSystem)
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"flatpak-builder", "flatpak"}, Pipe{}.Dependencies(nil))
}

func TestFlatpakArch(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"linuxamd64v1", "x86_64"},
		{"linuxarm64", "aarch64"},
		{"linux386", "i386"},
		{"linuxarm6", "arm"},
		{"linuxarm7", "arm"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			require.Equal(t, tt.want, flatpakArch(tt.key))
		})
	}
}

func TestIsValidArch(t *testing.T) {
	tests := []struct {
		arch string
		want bool
	}{
		{"x86_64", true},
		{"aarch64", true},
		{"arm", true},
		{"i386", true},
		{"mips", false},
		{"ppc64le", false},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			require.Equal(t, tt.want, isValidArch(tt.arch))
		})
	}
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	t.Helper()
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm"} {
			binPath := filepath.Join(dist, name)
			require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0o755))
			f, err := os.Create(binPath)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			switch goarch {
			case "arm":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   name,
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Goarm:  "6",
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})
			case "amd64":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    name,
					Path:    binPath,
					Goarch:  goarch,
					Goos:    goos,
					Goamd64: "v1",
					Type:    artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})
			default:
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   name,
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})
			}
		}
	}
}

func requireNoGerror(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		return
	}
	gerr, ok := errors.AsType[gerrors.ErrDetailed](err)
	require.True(tb, ok)
	require.NoError(tb, err, fmt.Sprintf("messages: %v, details: %v, output: %s", gerr.Messages(), maps.Collect(gerr.Details()), gerr.Output()))
}
