package sbom

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSBOMCatalogInvalidArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		SBOMs: []config.SBOM{{Artifacts: "foo"}},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to catalog: foo")
}

func TestSeveralSBOMsWithTheSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		SBOMs: []config.SBOM{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 sboms with the ID 'a', please fix your config")
}

func TestSkipCataloging(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})

	t.Run("skip SBOM cataloging", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{
					Artifacts: "all",
				},
			},
		}, testctx.Skip(skips.SBOM))

		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{
					Artifacts: "all",
				},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDisable(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{Disable: "false"},
			},
		})

		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{Disable: "true"},
			},
		})

		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})

	t.Run("enabled template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{Disable: `{{ eq .Env.SBOM_DISABLED "1" }}`},
			},
		}, testctx.WithEnv(map[string]string{"SBOM_DISABLED": "0"}))

		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("disabled template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{Disable: `{{ eq .Env.SBOM_DISABLED "1" }}`},
			},
		}, testctx.WithEnv(map[string]string{"SBOM_DISABLED": "1"}))

		testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	})

	t.Run("enabled invalid template", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			SBOMs: []config.SBOM{
				{Disable: "{{ .Invalid }}"},
			},
		})

		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func Test_subprocessDistPath(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name              string
		distDir           string
		pathRelativeToCwd string
		expects           string
	}{
		{
			name:              "relative dist with anchor",
			distDir:           "./dist",
			pathRelativeToCwd: "dist/my.sbom",
			expects:           "my.sbom",
		},
		{
			name:              "relative dist without anchor",
			distDir:           "dist",
			pathRelativeToCwd: "dist/my.sbom",
			expects:           "my.sbom",
		},
		{
			name:              "relative dist with nested resource",
			distDir:           "dist",
			pathRelativeToCwd: "dist/something/my.sbom",
			expects:           "something/my.sbom",
		},
		{
			name:              "absolute dist with nested resource",
			distDir:           filepath.Join(cwd, "dist/"),
			pathRelativeToCwd: "dist/something/my.sbom",
			expects:           "something/my.sbom",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := subprocessDistPath(test.distDir, test.pathRelativeToCwd)
			require.NoError(t, err)
			assert.Equal(t, filepath.ToSlash(test.expects), filepath.ToSlash(actual))
		})
	}
}

func Test_templateNames(t *testing.T) {
	art := artifact.Artifact{
		Name:   "name-it",
		Path:   "to/a/place",
		Goos:   "darwin",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "id-it",
			"Binary":         "binary-name",
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

	abs := func(path string) string {
		path, _ = filepath.Abs(path)
		return path
	}

	tests := []struct {
		name           string
		dist           string
		version        string
		cfg            config.SBOM
		artifact       artifact.Artifact
		expectedValues map[string]string
		expectedPaths  []string
	}{
		{
			name:     "default configuration",
			artifact: art,
			cfg:      config.SBOM{},
			dist:     abs("/somewhere/to/dist"),
			expectedPaths: []string{
				abs("/somewhere/to/dist/name-it.sbom.json"),
			},
			expectedValues: map[string]string{
				"artifact":   filepath.FromSlash("to/a/place"),
				"artifactID": "id-it",
				"document":   abs("/somewhere/to/dist/name-it.sbom.json"),
				"document0":  abs("/somewhere/to/dist/name-it.sbom.json"),
			},
		},
		{
			name:     "default configuration + relative dist",
			artifact: art,
			cfg:      config.SBOM{},
			dist:     "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/name-it.sbom.json"),
			},
			expectedValues: map[string]string{
				"artifact":   filepath.FromSlash("to/a/place"), // note: this is always relative to ${dist}
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/name-it.sbom.json"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/name-it.sbom.json"),
			},
		},
		{
			name: "custom document using $artifact",
			// note: this configuration is probably a misconfiguration since it is placing SBOMs within each bin
			// directory, however, it will behave as correctly as possible.
			artifact: art,
			cfg: config.SBOM{
				Documents: []string{
					// note: the artifact name is probably an incorrect value here since it can't express all attributes
					// of the binary (os, arch, etc), so builds with multiple architectures will create SBOMs with the
					// same name.
					"${artifact}.cdx.sbom.json",
				},
			},
			dist: "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom.json"),
			},
			expectedValues: map[string]string{
				"artifact":   filepath.FromSlash("to/a/place"),
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom.json"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom.json"),
			},
		},
		{
			name:     "custom document using build vars",
			artifact: art,
			cfg: config.SBOM{
				Documents: []string{
					"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.cdx.sbom.json",
				},
			},
			version: "1.0.0",
			dist:    "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
			},
			expectedValues: map[string]string{
				"artifact":   filepath.FromSlash("to/a/place"),
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
			},
		},
		{
			name:     "env vars with go templated options",
			artifact: art,
			cfg: config.SBOM{
				Documents: []string{
					"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.cdx.sbom.json",
				},
				Env: []string{
					"with-env-var=value",
					"custom-os={{ .Os }}-unique",
					"custom-arch={{ .Arch }}-unique",
				},
			},
			version: "1.0.0",
			dist:    "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
			},
			expectedValues: map[string]string{
				"artifact":     filepath.FromSlash("to/a/place"),
				"artifactID":   "id-it",
				"with-env-var": "value",
				"custom-os":    "darwin-unique",
				"custom-arch":  "amd64-unique",
				"document":     filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
				"document0":    filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom.json"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist: tt.dist,
			}, testctx.WithVersion(tt.version))

			cfg := tt.cfg
			require.NoError(t, setConfigDefaults(&cfg))

			var inputArgs []string
			var expectedArgs []string
			for key, value := range tt.expectedValues {
				inputArgs = append(inputArgs, fmt.Sprintf("${%s}", key))
				expectedArgs = append(expectedArgs, value)
			}
			cfg.Args = inputArgs

			actualArgs, actualEnvs, actualPaths, err := applyTemplate(ctx, cfg, &tt.artifact)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPaths, actualPaths, "paths differ")

			assert.Equal(t, expectedArgs, actualArgs, "arguments differ")

			actualEnv := make(map[string]string)
			for _, str := range actualEnvs {
				k, v, ok := strings.Cut(str, "=")
				require.True(t, ok)
				actualEnv[k] = v
			}

			for k, v := range tt.expectedValues {
				assert.Equal(t, v, actualEnv[k])
			}
		})
	}
}

func TestDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		SBOMs: []config.SBOM{
			{Cmd: "syft"},
			{Cmd: "foobar"},
		},
	})

	require.Equal(t, []string{"syft", "foobar"}, Pipe{}.Dependencies(ctx))
}
