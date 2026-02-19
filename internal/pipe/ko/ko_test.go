package ko

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

const registry1 = "localhost:5052/"

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Env: []string{
			"KO_DOCKER_REPO=" + registry1,
			"COSIGN_REPOSITORY=" + registry1,
			"LDFLAGS=foobar",
			"FLAGS=barfoo",
			"LE_ENV=test",
		},
		ProjectName: "test",
		Builds: []config.Build{
			{
				ID:  "test",
				Dir: ".",
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"{{.Env.LDFLAGS}}"},
					Flags:   []string{"{{.Env.FLAGS}}"},
					Env:     []string{"SOME_ENV={{.Env.LE_ENV}}"},
				},
			},
		},
		Kos: []config.Ko{
			{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, config.Ko{
		ID:           "test",
		Build:        "test",
		BaseImage:    chainguardStatic,
		Repositories: []string{registry1},
		Platforms:    []string{"linux/amd64"},
		SBOM:         "spdx",
		Tags:         []string{"latest"},
		WorkingDir:   ".",
		Ldflags:      []string{"{{.Env.LDFLAGS}}"},
		Flags:        []string{"{{.Env.FLAGS}}"},
		Env:          []string{"SOME_ENV={{.Env.LE_ENV}}"},
	}, ctx.Config.Kos[0])
}

func TestDefaultCycloneDX(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "test",
		Env:         []string{"KO_DOCKER_REPO=" + registry1},
		Kos: []config.Ko{
			{SBOM: "cyclonedx"},
		},
		Builds: []config.Build{
			{ID: "test"},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, ctx.Deprecated)
	require.Equal(t, "none", ctx.Config.Kos[0].SBOM)
}

func TestDefaultGoVersionM(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "test",
		Env:         []string{"KO_DOCKER_REPO=" + registry1},
		Kos: []config.Ko{
			{SBOM: "go.version-m"},
		},
		Builds: []config.Build{
			{ID: "test"},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, ctx.Deprecated)
	require.Equal(t, "none", ctx.Config.Kos[0].SBOM)
}

func TestDefaultNoImage(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "test",
		Builds: []config.Build{
			{
				ID: "test",
			},
		},
		Kos: []config.Ko{
			{},
		},
	})

	require.ErrorIs(t, Pipe{}.Default(ctx), errNoRepositories)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip ko set", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Kos: []config.Ko{{}},
		}, testctx.Skip(skips.Ko))

		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("skip no kos", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Kos: []config.Ko{{}},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPublishPipeNoMatchingBuild(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{
				ID: "doesnt matter",
			},
		},
		Kos: []config.Ko{
			{
				ID:    "default",
				Build: "wont match nothing",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), `no builds with id "wont match nothing"`)
}

func TestKoValidateMainPathIssue4382(t *testing.T) {
	// testing the validation of the main path directly to cover many cases
	require.NoError(t, validateMainPath(""))
	require.NoError(t, validateMainPath("."))
	require.NoError(t, validateMainPath("./..."))
	require.NoError(t, validateMainPath("./app"))
	require.NoError(t, validateMainPath("../../../..."))
	require.NoError(t, validateMainPath("../../app/"))
	require.NoError(t, validateMainPath("./testdata/app/main"))
	require.NoError(t, validateMainPath("./testdata/app/folder.with.dots"))

	require.ErrorIs(t, validateMainPath("app/"), errInvalidMainPath)
	require.ErrorIs(t, validateMainPath("/src/"), errInvalidMainPath)
	require.ErrorIs(t, validateMainPath("/src/app"), errInvalidMainPath)
	require.ErrorIs(t, validateMainPath("./testdata/app/main.go"), errInvalidMainGoPath)

	// testing with real context
	ctxOk := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{
				ID:   "foo",
				Main: "./...",
			},
		},
		Kos: []config.Ko{
			{
				ID:         "default",
				Build:      "foo",
				Repository: "fakerepo",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctxOk))

	ctxWithInvalidMainPath := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{
				ID:   "foo",
				Main: "/some/non/relative/path",
			},
		},
		Kos: []config.Ko{
			{
				ID:         "default",
				Build:      "foo",
				Repository: "fakerepo",
			},
		},
	})

	require.ErrorIs(t, Pipe{}.Default(ctxWithInvalidMainPath), errInvalidMainPath)
}

func TestApplyTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		foo, err := applyTemplate(testctx.WrapWithCfg(t.Context(), config.Project{
			Env: []string{"FOO=bar"},
		}),

			[]string{"{{ .Env.FOO }}"})
		require.NoError(t, err)
		require.Equal(t, []string{"bar"}, foo)
	})
	t.Run("error", func(t *testing.T) {
		_, err := applyTemplate(testctx.Wrap(t.Context()), []string{"{{ .Nope}}"})
		require.Error(t, err)
	})
}

func TestGetLocalDomain(t *testing.T) {
	t.Run("default local domain", func(t *testing.T) {
		ko := config.Ko{}
		got := getLocalDomain(ko)
		require.Equal(t, "goreleaser.ko.local", got)
	})

	t.Run("custom local domain", func(t *testing.T) {
		ko := config.Ko{LocalDomain: "custom.domain"}
		got := getLocalDomain(ko)
		require.Equal(t, "custom.domain", got)
	})
}
