package ko

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/goreleaser/goreleaser/internal/testctx"
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
	ctx := testctx.NewWithCfg(config.Project{
		Env: []string{
			"KO_DOCKER_REPO=" + registry,
			"COSIGN_REPOSITORY=" + registry,
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
		ID:         "test",
		Build:      "test",
		BaseImage:  chainguardStatic,
		Repository: registry,
		Platforms:  []string{"linux/amd64"},
		SBOM:       "spdx",
		Tags:       []string{"latest"},
		WorkingDir: ".",
		Ldflags:    []string{"{{.Env.LDFLAGS}}"},
		Flags:      []string{"{{.Env.FLAGS}}"},
		Env:        []string{"SOME_ENV={{.Env.LE_ENV}}"},
	}, ctx.Config.Kos[0])
}

func TestDefaultNoImage(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	require.ErrorIs(t, Pipe{}.Default(ctx), errNoRepository)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip ko set", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Kos: []config.Ko{{}},
		})
		ctx.SkipKo = true
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("skip no kos", func(t *testing.T) {
		ctx := testctx.New()
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Kos: []config.Ko{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestPublishPipeNoMatchingBuild(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
		Name               string
		SBOM               string
		BaseImage          string
		Labels             map[string]string
		ExpectedLabels     map[string]string
		Platforms          []string
		CreationTime       string
		KoDataCreationTime string
	}{
		{
			// Must be first as others add an SBOM for the same image
			Name: "sbom-none",
			SBOM: "none",
		},
		{
			Name: "sbom-spdx",
			SBOM: "spdx",
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
		{
			Name:           "labels",
			Labels:         map[string]string{"foo": "bar", "project": "{{.ProjectName}}"},
			ExpectedLabels: map[string]string{"foo": "bar", "project": "test"},
		},
		{
			Name:         "creation-time",
			CreationTime: "1672531200",
		},
		{
			Name:               "kodata-creation-time",
			KoDataCreationTime: "1672531200",
		},
	}

	repository := fmt.Sprintf("%sgoreleasertest/testapp", registry)

	for _, table := range table {
		t.Run(table.Name, func(t *testing.T) {
			ctx := testctx.NewWithCfg(config.Project{
				ProjectName: "test",
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
						ID:                 "default",
						Build:              "foo",
						WorkingDir:         "./testdata/app/",
						BaseImage:          table.BaseImage,
						Repository:         repository,
						Labels:             table.Labels,
						Platforms:          table.Platforms,
						Tags:               []string{table.Name},
						CreationTime:       table.CreationTime,
						KoDataCreationTime: table.KoDataCreationTime,
						SBOM:               table.SBOM,
						Bare:               true,
					},
				},
			})

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Publish(ctx))

			ref, err := name.ParseReference(
				fmt.Sprintf("%s:%s", repository, table.Name),
				name.Insecure,
			)
			require.NoError(t, err)

			index, err := remote.Index(ref)
			if len(table.Platforms) > 1 {
				require.NoError(t, err)
				imf, err := index.IndexManifest()
				require.NoError(t, err)

				platforms := make([]string, 0, len(imf.Manifests))
				for _, mf := range imf.Manifests {
					platforms = append(platforms, mf.Platform.String())
				}
				require.ElementsMatch(t, table.Platforms, platforms)
			} else {
				require.Error(t, err)
			}

			image, err := remote.Image(ref)
			require.NoError(t, err)

			digest, err := image.Digest()
			require.NoError(t, err)

			sbomRef, err := name.ParseReference(
				fmt.Sprintf(
					"%s:%s.sbom",
					repository,
					strings.Replace(digest.String(), ":", "-", 1),
				),
				name.Insecure,
			)
			require.NoError(t, err)

			sbom, err := remote.Image(sbomRef)
			if table.SBOM == "none" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				layers, err := sbom.Layers()
				require.NoError(t, err)
				require.NotEmpty(t, layers)

				mediaType, err := layers[0].MediaType()
				require.NoError(t, err)

				switch table.SBOM {
				case "spdx", "":
					require.Equal(t, "spdx+json", string(mediaType))
				case "cyclonedx":
					require.Equal(t, "application/vnd.cyclonedx+json", string(mediaType))
				case "go.version-m":
					require.Equal(t, "application/vnd.go.version-m", string(mediaType))
				default:
					require.Fail(t, "unknown SBOM type", table.SBOM)
				}
			}

			configFile, err := image.ConfigFile()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(configFile.History), 3)

			require.Equal(t, table.ExpectedLabels, configFile.Config.Labels)

			var creationTime time.Time
			if table.CreationTime != "" {
				ct, err := strconv.ParseInt(table.CreationTime, 10, 64)
				require.NoError(t, err)
				creationTime = time.Unix(ct, 0).UTC()

				require.Equal(t, creationTime, configFile.Created.Time)
			}
			require.Equal(t, creationTime, configFile.History[len(configFile.History)-1].Created.Time)

			var koDataCreationTime time.Time
			if table.KoDataCreationTime != "" {
				kdct, err := strconv.ParseInt(table.KoDataCreationTime, 10, 64)
				require.NoError(t, err)
				koDataCreationTime = time.Unix(kdct, 0).UTC()
			}
			require.Equal(t, koDataCreationTime, configFile.History[len(configFile.History)-2].Created.Time)
		})
	}
}

func TestPublishPipeError(t *testing.T) {
	makeCtx := func() *context.Context {
		return testctx.NewWithCfg(config.Project{
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
					WorkingDir: "./testdata/app/",
					Repository: "fakerepo:8080/",
					Tags:       []string{"latest", "{{.Tag}}"},
				},
			},
		}, testctx.WithCurrentTag("v1.0.0"))
	}

	t.Run("invalid base image", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].BaseImage = "not a valid image hopefully"
		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Publish(ctx), `build: could not parse reference: not a valid image hopefully`)
	})

	t.Run("invalid label tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].Labels = map[string]string{"nope": "{{.Nope}}"}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
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

	t.Run("invalid tags tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].Tags = []string{"{{.Nope}}"}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})

	t.Run("invalid creation time", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].CreationTime = "nope"
		require.NoError(t, Pipe{}.Default(ctx))
		err := Pipe{}.Publish(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), `strconv.ParseInt: parsing "nope": invalid syntax`)
	})

	t.Run("invalid creation time tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].CreationTime = "{{.Nope}}"
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
	})

	t.Run("invalid kodata creation time", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].KoDataCreationTime = "nope"
		require.NoError(t, Pipe{}.Default(ctx))
		err := Pipe{}.Publish(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), `strconv.ParseInt: parsing "nope": invalid syntax`)
	})

	t.Run("invalid kodata creation time tmpl", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.Kos[0].KoDataCreationTime = "{{.Nope}}"
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
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
		err := Pipe{}.Publish(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), `publish: writing sbom: Get "https://fakerepo:8080/v2/": dial tcp:`)
	})
}

func TestApplyTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		foo, err := applyTemplate(testctx.NewWithCfg(config.Project{
			Env: []string{"FOO=bar"},
		}), []string{"{{ .Env.FOO }}"})
		require.NoError(t, err)
		require.Equal(t, []string{"bar"}, foo)
	})
	t.Run("error", func(t *testing.T) {
		_, err := applyTemplate(testctx.New(), []string{"{{ .Nope}}"})
		require.Error(t, err)
	})
}
