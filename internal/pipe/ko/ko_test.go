package ko

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

const (
	registry1Port = "5052"
	registry1     = "localhost:5052/"
	registry2Port = "5053"
	registry2     = "localhost:5053/"
)

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	require.ErrorIs(t, Pipe{}.Default(ctx), errNoRepositories)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip ko set", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Kos: []config.Ko{{}},
		}, testctx.Skip(skips.Ko))
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
	testlib.SkipIfWindows(t, "ko doesn't work in windows")
	testlib.CheckDocker(t)
	testlib.StartRegistry(t, "ko_registry1", registry1Port)
	testlib.StartRegistry(t, "ko_registry2", registry2Port)

	chainguardStaticLabels := map[string]string{
		"dev.chainguard.package.main":      "",
		"org.opencontainers.image.authors": "Chainguard Team https://www.chainguard.dev/",
		"org.opencontainers.image.source":  "https://github.com/chainguard-images/images/tree/main/images/static",
		"org.opencontainers.image.url":     "https://images.chainguard.dev/directory/image/static/overview",
		"org.opencontainers.image.vendor":  "Chainguard",
		"org.opencontainers.image.created": ".*",
	}
	baseImageAnnotations := map[string]string{
		"org.opencontainers.image.base.name":   ".*",
		"org.opencontainers.image.base.digest": ".*",
	}

	table := []struct {
		Name                string
		SBOM                string
		SBOMDirectory       string
		BaseImage           string
		Labels              map[string]string
		ExpectedLabels      map[string]string
		Annotations         map[string]string
		ExpectedAnnotations map[string]string
		User                string
		Platforms           []string
		Tags                []string
		CreationTime        string
		KoDataCreationTime  string
	}{
		{
			// Must be first as others add an SBOM for the same image
			Name:          "sbom-none",
			SBOM:          "none",
			SBOMDirectory: "",
		},
		{
			Name: "sbom-spdx",
			SBOM: "spdx",
		},
		{
			Name:          "sbom-spdx-with-dir",
			SBOM:          "spdx",
			SBOMDirectory: "testdata/app/",
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
			Name:                "annotations",
			Annotations:         map[string]string{"foo": "bar", "project": "{{.ProjectName}}"},
			ExpectedAnnotations: map[string]string{"foo": "bar", "project": "test"},
		},
		{
			Name: "user",
			User: "1234:1234",
		},
		{
			Name:         "creation-time",
			CreationTime: "1672531200",
		},
		{
			Name:               "kodata-creation-time",
			KoDataCreationTime: "1672531200",
		},
		{
			Name: "tag-templates",
			Tags: []string{
				"{{if not .Prerelease }}{{.Version}}{{ end }}",
				"   ", // empty
			},
		},
		{
			Name: "tag-template-eval-empty",
			Tags: []string{
				"{{.Version}}",
				"{{if .Prerelease }}latest{{ end }}",
			},
		},
	}

	repositories := []string{
		fmt.Sprintf("%sgoreleasertest/testapp", registry1),
		fmt.Sprintf("%sgoreleasertest/testapp", registry2),
	}

	for _, table := range table {
		t.Run(table.Name, func(t *testing.T) {
			if len(table.Tags) == 0 {
				table.Tags = []string{table.Name}
			}
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
						Repositories:       repositories,
						Labels:             table.Labels,
						Annotations:        table.Annotations,
						User:               table.User,
						Platforms:          table.Platforms,
						Tags:               table.Tags,
						CreationTime:       table.CreationTime,
						KoDataCreationTime: table.KoDataCreationTime,
						SBOM:               table.SBOM,
						SBOMDirectory:      table.SBOMDirectory,
						Bare:               true,
					},
				},
			}, testctx.WithVersion("1.2.0"))

			if table.BaseImage == "" {
				if table.User == "" {
					table.User = "65532"
				}
				table.ExpectedLabels = mergeMaps(table.ExpectedLabels, chainguardStaticLabels)
			}
			table.ExpectedAnnotations = mergeMaps(table.ExpectedAnnotations, baseImageAnnotations)

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Publish(ctx))

			manifests := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerManifest)).List()
			require.Len(t, manifests, 2) // both registries
			require.NotEmpty(t, manifests[0].Name)
			require.Equal(t, manifests[0].Name, manifests[0].Path)
			require.NotEmpty(t, manifests[0].Extra[artifact.ExtraDigest])
			require.Equal(t, "default", manifests[0].Extra[artifact.ExtraID])

			tags, err := applyTemplate(ctx, table.Tags)
			require.NoError(t, err)
			tags = removeEmpty(tags)
			require.Len(t, tags, 1)

			repository := repositories[0]
			ref, err := name.ParseReference(
				fmt.Sprintf("%s:latest", repository),
				name.Insecure,
			)
			require.NoError(t, err)
			_, err = remote.Index(ref)
			require.Error(t, err) // latest should not exist

			ref, err = name.ParseReference(
				fmt.Sprintf("%s:%s", repository, tags[0]),
				name.Insecure,
			)
			require.NoError(t, err)

			repository2 := repositories[1]
			_, err = name.ParseReference(
				fmt.Sprintf("%s:%s", repository2, tags[0]),
				name.Insecure,
			)
			require.NoError(t, err)

			index, err := remote.Index(ref)
			if len(table.Platforms) > 1 {
				require.NoError(t, err)
				imf, err := index.IndexManifest()
				require.NoError(t, err)

				compareMaps(t, table.ExpectedAnnotations, imf.Annotations)

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
					require.Equal(t, "text/spdx+json", string(mediaType))
				default:
					require.Fail(t, "unknown SBOM type", table.SBOM)
				}
			}

			mf, err := image.Manifest()
			require.NoError(t, err)

			expectedAnnotations := table.ExpectedAnnotations
			if table.BaseImage == "" {
				expectedAnnotations = mergeMaps(
					expectedAnnotations,
					chainguardStaticLabels,
				)
			}
			compareMaps(t, expectedAnnotations, mf.Annotations)

			configFile, err := image.ConfigFile()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(configFile.History), 3)

			compareMaps(t, table.ExpectedLabels, configFile.Config.Labels)
			require.Equal(t, table.User, configFile.Config.User)

			var creationTime time.Time
			if table.CreationTime != "" {
				ct, err := strconv.ParseInt(table.CreationTime, 10, 64)
				require.NoError(t, err)
				creationTime = time.Unix(ct, 0).UTC()

				require.Equal(t, creationTime, configFile.Created.UTC())
			}
			require.Equal(t, creationTime, configFile.History[len(configFile.History)-1].Created.UTC())

			var koDataCreationTime time.Time
			if table.KoDataCreationTime != "" {
				kdct, err := strconv.ParseInt(table.KoDataCreationTime, 10, 64)
				require.NoError(t, err)
				koDataCreationTime = time.Unix(kdct, 0).UTC()
			}
			require.Equal(t, koDataCreationTime, configFile.History[len(configFile.History)-2].Created.UTC())
		})
	}
}

func TestSnapshot(t *testing.T) {
	testlib.SkipIfWindows(t, "ko doesn't work in windows")
	testlib.CheckDocker(t)
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
				ID:         "default",
				Build:      "foo",
				Repository: "testimage",
				WorkingDir: "./testdata/app/",
				Tags:       []string{"latest"},
			},
		},
	}, testctx.WithVersion("1.2.0"), testctx.Snapshot)

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	manifests := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerManifest)).List()
	require.Len(t, manifests, 1)
	require.NotEmpty(t, manifests[0].Name)
	require.Equal(t, manifests[0].Name, manifests[0].Path)
	require.NotEmpty(t, manifests[0].Extra[artifact.ExtraDigest])
	require.Equal(t, "default", manifests[0].Extra[artifact.ExtraID])
}

func TestDisable(t *testing.T) {
	testlib.SkipIfWindows(t, "ko doesn't work in windows")
	testlib.CheckDocker(t)
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
				ID:         "disabled",
				Build:      "foo",
				Disable:    "{{ not (isEnvSet \"FOO\")}}",
				Repository: "NOPE",
			},
			{
				ID:         "default",
				Build:      "foo",
				Repository: "testimage",
				WorkingDir: "./testdata/app/",
				Tags:       []string{"latest"},
			},
		},
	}, testctx.WithVersion("1.2.0"), testctx.Snapshot)

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.True(t, pipe.IsSkip(err))

	manifests := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerManifest)).List()
	require.Len(t, manifests, 1)
	require.NotEmpty(t, manifests[0].Name)
	require.Equal(t, manifests[0].Name, manifests[0].Path)
	require.NotEmpty(t, manifests[0].Extra[artifact.ExtraDigest])
	require.Equal(t, "default", manifests[0].Extra[artifact.ExtraID])
}

func TestDisableInvalidTemplate(t *testing.T) {
	testlib.SkipIfWindows(t, "ko doesn't work in windows")
	testlib.CheckDocker(t)
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "test",
		Builds:      []config.Build{{ID: "foo"}},
		Kos: []config.Ko{
			{
				ID:         "disabled",
				Build:      "foo",
				Disable:    "{{ .nope }}",
				Repository: "NOPE",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Publish(ctx))
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
	ctxOk := testctx.NewWithCfg(config.Project{
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

	ctxWithInvalidMainPath := testctx.NewWithCfg(config.Project{
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
		require.EqualError(t, Pipe{}.Publish(ctx), `build: fetching base image: could not parse reference: not a valid image hopefully`)
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
		require.EqualError(
			t, Pipe{}.Publish(ctx),
			"build: build: go build: exit status 1: pattern ./...: directory prefix . does not contain main module or its selected dependencies\n",
		)
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
		require.ErrorContains(t, err, `strconv.ParseInt: parsing "nope": invalid syntax`)
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
		require.ErrorContains(t, err, `strconv.ParseInt: parsing "nope": invalid syntax`)
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
		require.Contains(t, err.Error(), `Get "https://fakerepo:8080/v2/": dial tcp:`)
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

func mergeMaps(ms ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, m := range ms {
		if m != nil {
			maps.Copy(result, m)
		}
	}
	return result
}

func compareMaps(t *testing.T, expected, actual map[string]string) {
	t.Helper()
	require.Len(t, actual, len(expected), "expected: %v", expected)
	for k, v := range expected {
		got, ok := actual[k]
		require.True(t, ok, "missing key: %s", k)
		require.Regexp(t, v, got, "key: %s", k)
	}
}
