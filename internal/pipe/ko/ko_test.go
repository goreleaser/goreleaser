package ko

import (
	"fmt"
	"testing"

	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

const (
	registryPort = "5052"
	registry     = "localhost:5052/"
)

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{
		Env: []string{
			"KO_DOCKER_REPO=" + registry,
			"COSIGN_REPOSITORY=" + registry,
			"LDFLAGS=foobar",
			"FLAGS=barfoo",
			"LE_ENV=test",
		},
		Builds: []config.Build{
			{
				ID: "default",
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
		ID:               "default",
		Build:            "default",
		BaseImage:        chainguardStatic,
		Repository:       registry,
		CosignRepository: registry,
		Platforms:        []string{"linux/amd64"},
		SBOM:             "spdx",
		Tags:             []string{"latest"},
		WorkingDir:       ".",
		Ldflags:          []string{"{{.Env.LDFLAGS}}"},
		Flags:            []string{"{{.Env.FLAGS}}"},
		Env:              []string{"SOME_ENV={{.Env.LE_ENV}}"},
	}, ctx.Config.Kos[0])
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip ko set", func(t *testing.T) {
		ctx := context.New(config.Project{
			Kos: []config.Ko{{}},
		})
		ctx.SkipKo = true
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("skip no kos", func(t *testing.T) {
		ctx := context.New(config.Project{})
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Kos: []config.Ko{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPublishPipeNoMatchingBuild(t *testing.T) {
	ctx := context.New(config.Project{
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

func TestPublishPipeSuccess(t *testing.T) {
	testlib.StartRegistry(t, "ko_registry", registryPort)

	table := []struct {
		Name      string
		SBOM      string
		BaseImage string
		Platforms []string
	}{
		{
			Name: "sbom-spdx",
			SBOM: "spdx",
		},
		{
			Name: "sbom-none",
			SBOM: "none",
		},
		{
			Name: "sbom-cyclonedx",
			SBOM: "cyclonedx",
		},
		{
			Name: "sbom-go.version-m",
			SBOM: "go.version-m",
		},
		{
			Name:      "base-image-is-not-index",
			BaseImage: "alpine:latest@sha256:c0d488a800e4127c334ad20d61d7bc21b4097540327217dfab52262adc02380c",
		},
		{
			Name:      "multiple-platforms",
			Platforms: []string{"linux/amd64", "linux/arm64"},
		},
	}

	for _, table := range table {
		t.Run(table.Name, func(t *testing.T) {
			ctx := context.New(config.Project{
				Builds: []config.Build{
					{
						ID: "foo",
						BuildDetails: config.BuildDetails{
							Ldflags: []string{"-s", "-w"},
							Flags:   []string{"-tags", "netgo"},
							Env:     []string{"GOCACHE=" + t.TempDir()},
						},
					},
				},
				Kos: []config.Ko{
					{
						ID:         "default",
						Build:      "foo",
						WorkingDir: "./testdata/app/",
						BaseImage:  table.BaseImage,
						Repository: fmt.Sprintf("%s/goreleasertest", registry),
						Platforms:  table.Platforms,
						Tags:       []string{table.Name},
						SBOM:       table.SBOM,
					},
				},
			})

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Publish(ctx))
		})
	}
}

func TestPublishPipeError(t *testing.T) {
	makeCtx := func() *context.Context {
		return context.New(config.Project{
			Builds: []config.Build{
				{ID: "foo"},
			},
			Kos: []config.Ko{
				{
					ID:         "default",
					Build:      "foo",
					WorkingDir: "./testdata/app/",
					Repository: "fakerepo:8080/",
				},
			},
		})
	}

	t.Run("invalid base image", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].BaseImage = "not a valid image hopefully"
		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Publish(ctx), `build: could not parse reference: not a valid image hopefully`)
	})

	t.Run("invalid sbom", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].SBOM = "nope"
		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Publish(ctx), `makeBuilder: unknown sbom type: "nope"`)
	})

	t.Run("invalid build", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].WorkingDir = t.TempDir()
		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Publish(ctx), `build: exit status 1`)
	})

	t.Run("invalid env tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Builds[0].Env = []string{"{{.Nope}}"}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})

	t.Run("invalid ldflags tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Builds[0].Ldflags = []string{"{{.Nope}}"}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})

	t.Run("invalid flags tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Builds[0].Flags = []string{"{{.Nope}}"}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})

	t.Run("publish fail", func(t *testing.T) {
		ctx := makeCtx()
		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Publish(ctx), `publish: writing sbom: Get "https://fakerepo:8080/v2/": dial tcp: lookup fakerepo: Temporary failure in name resolution`)
	})
}

func TestApplyTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		foo, err := applyTemplate(context.New(config.Project{
			Env: []string{"FOO=bar"},
		}), []string{"{{ .Env.FOO }}"})
		require.NoError(t, err)
		require.Equal(t, []string{"bar"}, foo)
	})
	t.Run("error", func(t *testing.T) {
		_, err := applyTemplate(context.New(config.Project{}), []string{"{{ .Nope}}"})
		require.Error(t, err)
	})
}
