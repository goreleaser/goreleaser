package snapcraft

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
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

func TestRunPipeMissingInfo(t *testing.T) {
	for eerr, snap := range map[error]config.Snapcraft{
		ErrNoSummary: {
			Description: "dummy desc",
		},
		ErrNoDescription: {
			Summary: "dummy summary",
		},
		pipe.Skip("no summary nor description were provided"): {},
	} {
		t.Run(fmt.Sprintf("testing if %v happens", eerr), func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Snapcrafts: []config.Snapcraft{
					snap,
				},
			})

			require.Equal(t, eerr, Pipe{}.Run(ctx))
		})
	}
}

func TestNoSnapcraftInPath(t *testing.T) {
	t.Setenv("PATH", "")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				Summary:     "dummy",
				Description: "dummy",
			},
		},
	})

	require.EqualError(t, Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Snapcrafts: []config.Snapcraft{{
			Description: "hi",
			Summary:     "hi",
			Builds:      []string{"a"},
		}},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Snapcrafts[0].NameTemplate)
	require.Equal(t, []string{"edge", "beta", "candidate", "stable"}, ctx.Config.Snapcrafts[0].ChannelTemplates)
	require.Equal(t, "stable", ctx.Config.Snapcrafts[0].Grade)
	require.Equal(t, "strict", ctx.Config.Snapcrafts[0].Confinement)
	require.Equal(t, []string{"a"}, ctx.Config.Snapcrafts[0].IDs)
	require.True(t, ctx.Deprecated)
}

func TestDefaultNoDescription(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{{ID: "foo"}},
		Snapcrafts: []config.Snapcraft{{
			Summary: "hi",
		}},
	})

	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultNoSummary(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{{ID: "foo"}},
		Snapcrafts: []config.Snapcraft{{
			Description: "hi",
		}},
	})

	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultGradeTmpl(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Env: []string{"Grade=devel"},
		Snapcrafts: []config.Snapcraft{
			{
				Grade:       "{{.Env.Grade}}",
				Description: "hi",
				Summary:     "hi",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Snapcrafts[0].NameTemplate)
	require.Equal(t, []string{"edge", "beta"}, ctx.Config.Snapcrafts[0].ChannelTemplates)
	require.Equal(t, "devel", ctx.Config.Snapcrafts[0].Grade)
}

func TestDefaultGradeTmplError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds:     []config.Build{{ID: "foo"}},
		Snapcrafts: []config.Snapcraft{{Grade: "{{.Env.Grade}}"}},
	})

	testlib.RequireTemplateError(t, Pipe{}.Default(ctx))
}

func TestPublish(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
		Extra: map[string]any{
			releasesExtra: []string{"stable", "candidate"},
		},
	})
	err := Pipe{}.Publish(ctx)
	require.ErrorContains(t, err, "failed to push nope.snap package")
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				ID:           "devel",
				NameTemplate: "foo",
				Grade:        "devel",
				Description:  "hi",
				Summary:      "hi",
			},
			{
				ID:           "stable",
				NameTemplate: "bar",
				Grade:        "stable",
				Description:  "hi",
				Summary:      "hi",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Snapcrafts[0].NameTemplate)
	require.Equal(t, []string{"edge", "beta"}, ctx.Config.Snapcrafts[0].ChannelTemplates)
	require.Equal(t, []string{"edge", "beta", "candidate", "stable"}, ctx.Config.Snapcrafts[1].ChannelTemplates)
}

func Test_processChannelsTemplates(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Builds: []config.Build{
				{
					ID: "default",
				},
			},
			Snapcrafts: []config.Snapcraft{
				{
					Name:        "mybin",
					Description: "hi",
					Summary:     "hi",
					ChannelTemplates: []string{
						"{{.Major}}.{{.Minor}}/stable",
						"stable",
					},
				},
			},
			Env: []string{"FOO=123"},
		},
		testctx.WithCommit("a1b2c3d4"),
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithSemver(1, 0, 0, ""),
		testctx.WithVersion("1.0.0"))

	require.NoError(t, Pipe{}.Default(ctx))
	snap := ctx.Config.Snapcrafts[0]
	require.Equal(t, "mybin", snap.Name)

	channels, err := processChannelsTemplates(ctx, snap)
	require.NoError(t, err)
	require.Equal(t, []string{
		"1.0/stable",
		"stable",
	}, channels)
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
					Name:   "subdir/" + name,
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
					Name:    "subdir/" + name,
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
					Name:   "subdir/" + name,
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

func TestSeveralSnapssWithTheSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				ID:          "a",
				Description: "hi",
				Summary:     "hi",
			},
			{
				ID:          "a",
				Description: "hi",
				Summary:     "hi",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 snapcrafts with the ID 'a', please fix your config")
}

func Test_isValidArch(t *testing.T) {
	tests := []struct {
		arch string
		want bool
	}{
		{"s390x", true},
		{"ppc64el", true},
		{"arm64", true},
		{"armhf", true},
		{"i386", true},
		{"mips", false},
		{"armel", false},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			require.Equal(t, tt.want, isValidArch(tt.arch))
		})
	}
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Snapcrafts: []config.Snapcraft{
				{},
			},
		}, testctx.Skip(skips.Snapcraft))

		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Snapcrafts: []config.Snapcraft{
				{},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Snapcrafts: []config.Snapcraft{
			{},
		},
	})

	require.Equal(t, []string{"snapcraft"}, Pipe{}.Dependencies(ctx))
}

func requireEqualFileContents(tb testing.TB, a, b string) {
	tb.Helper()
	eq, err := gio.EqualFileContents(a, b)
	require.NoError(tb, err)
	require.True(tb, eq, "%s != %s", a, b)
}
