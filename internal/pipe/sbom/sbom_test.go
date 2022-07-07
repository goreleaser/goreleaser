package sbom

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSBOMCatalogDefault(t *testing.T) {
	defaultArgs := []string{"$artifact", "--file", "$document", "--output", "spdx-json"}
	defaultSboms := []string{
		"{{ .ArtifactName }}.sbom",
	}
	defaultCmd := "syft"
	tests := []struct {
		configs  []config.SBOM
		artifact string
		cmd      string
		sboms    []string
		args     []string
		env      []string
		err      bool
	}{
		{
			configs: []config.SBOM{
				{
					// empty
				},
			},
			artifact: "archive",
			cmd:      defaultCmd,
			sboms:    defaultSboms,
			args:     defaultArgs,
			env: []string{
				"SYFT_FILE_METADATA_CATALOGER_ENABLED=true",
			},
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "package",
				},
			},
			artifact: "package",
			cmd:      defaultCmd,
			sboms:    defaultSboms,
			args:     defaultArgs,
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "archive",
				},
			},
			artifact: "archive",
			cmd:      defaultCmd,
			sboms:    defaultSboms,
			args:     defaultArgs,
			env: []string{
				"SYFT_FILE_METADATA_CATALOGER_ENABLED=true",
			},
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "archive",
					Env: []string{
						"something=something-else",
					},
				},
			},
			artifact: "archive",
			cmd:      defaultCmd,
			sboms:    defaultSboms,
			args:     defaultArgs,
			env: []string{
				"something=something-else",
			},
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "any",
				},
			},
			artifact: "any",
			cmd:      defaultCmd,
			sboms:    []string{},
			args:     defaultArgs,
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "binary",
				},
			},
			artifact: "binary",
			cmd:      defaultCmd,
			sboms:    []string{"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom"},
			args:     defaultArgs,
		},
		{
			configs: []config.SBOM{
				{
					Artifacts: "source",
				},
			},
			artifact: "source",
			cmd:      defaultCmd,
			sboms:    defaultSboms,
			args:     defaultArgs,
			env: []string{
				"SYFT_FILE_METADATA_CATALOGER_ENABLED=true",
			},
		},
		{
			// multiple documents are not allowed when artifacts != "any"
			configs: []config.SBOM{
				{
					Artifacts: "binary",
					Documents: []string{
						"doc1",
						"doc2",
					},
				},
			},
			err: true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("artifact=%q", test.configs[0].Artifacts), func(t *testing.T) {
			ctx := &context.Context{
				Config: config.Project{
					SBOMs: test.configs,
				},
			}
			err := Pipe{}.Default(ctx)
			if test.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, ctx.Config.SBOMs[0].Cmd, test.cmd)
			require.Equal(t, ctx.Config.SBOMs[0].Documents, test.sboms)
			require.Equal(t, ctx.Config.SBOMs[0].Args, test.args)
			require.Equal(t, ctx.Config.SBOMs[0].Env, test.env)
			require.Equal(t, ctx.Config.SBOMs[0].Artifacts, test.artifact)
		})
	}
}

func TestSBOMCatalogInvalidArtifacts(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.SBOMs = []config.SBOM{
		{Artifacts: "foo"},
	}
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to catalog: foo")
}

func TestSeveralSBOMsWithTheSameID(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			SBOMs: []config.SBOM{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 sboms with the ID 'a', please fix your config")
}

func TestSkipCataloging(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("skip SBOM cataloging", func(t *testing.T) {
		ctx := context.New(config.Project{
			SBOMs: []config.SBOM{
				{
					Artifacts: "all",
				},
			},
		})
		ctx.SkipSBOMCataloging = true
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			SBOMs: []config.SBOM{
				{
					Artifacts: "all",
				},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestSBOMCatalogArtifacts(t *testing.T) {
	tests := []struct {
		desc           string
		ctx            *context.Context
		sbomPaths      []string
		sbomNames      []string
		expectedErrMsg string
	}{
		{
			desc:           "catalog errors",
			expectedErrMsg: "cataloging artifacts: exit failed",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "binary",
							Cmd:       "exit",
							Args:      []string{"1"},
						},
					},
				},
			),
		},
		{
			desc:           "invalid args template",
			expectedErrMsg: `cataloging artifacts failed: arg "${FOO}-{{ .foo }{{}}{": invalid template: template: tmpl:1: unexpected "}" in operand`,
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "binary",
							Cmd:       "exit",
							Args:      []string{"${FOO}-{{ .foo }{{}}{"},
						},
					},
					Env: []string{
						"FOO=BAR",
					},
				},
			),
		},
		{
			desc: "catalog source archives",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{Artifacts: "source"},
					},
				},
			),
			sbomPaths: []string{"artifact5.tar.gz.sbom"},
			sbomNames: []string{"artifact5.tar.gz.sbom"},
		},
		{
			desc: "catalog archives",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{Artifacts: "archive"},
					},
				},
			),
			sbomPaths: []string{"artifact1.sbom", "artifact2.sbom"},
			sbomNames: []string{"artifact1.sbom", "artifact2.sbom"},
		},
		{
			desc: "catalog linux packages",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{Artifacts: "package"},
					},
				},
			),
			sbomPaths: []string{"package1.deb.sbom"},
			sbomNames: []string{"package1.deb.sbom"},
		},
		{
			desc: "catalog binaries",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{Artifacts: "binary"},
					},
				},
			),
			sbomPaths: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom",
				"artifact4-name_1.2.2_linux_amd64.sbom",
			},
			sbomNames: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom",
				"artifact4-name_1.2.2_linux_amd64.sbom",
			},
		},
		{
			desc: "manual cataloging",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "any",
							Args: []string{
								"--file",
								"$document0",
								"--output",
								"spdx-json",
								"artifact5.tar.gz",
							},
							Documents: []string{
								"final.sbom",
							},
						},
					},
				},
			),
			sbomPaths: []string{"final.sbom"},
			sbomNames: []string{"final.sbom"},
		},
		{
			desc: "multiple SBOM configs",
			ctx: context.New(
				config.Project{
					Env: []string{
						"SBOM_SUFFIX=s2-ish",
					},
					SBOMs: []config.SBOM{
						{
							ID:        "s1",
							Artifacts: "binary",
						},
						{
							ID:        "s2",
							Artifacts: "archive",
							Documents: []string{"{{ .ArtifactName }}.{{ .Env.SBOM_SUFFIX }}.sbom"},
						},
					},
				},
			),
			sbomPaths: []string{
				"artifact1.s2-ish.sbom",
				"artifact2.s2-ish.sbom",
				"artifact3-name_1.2.2_linux_amd64.sbom",
				"artifact4-name_1.2.2_linux_amd64.sbom",
			},
			sbomNames: []string{
				"artifact1.s2-ish.sbom",
				"artifact2.s2-ish.sbom",
				"artifact3-name_1.2.2_linux_amd64.sbom",
				"artifact4-name_1.2.2_linux_amd64.sbom",
			},
		},
		{
			desc: "catalog artifacts with filtered by ID",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "binary",
							IDs:       []string{"foo"},
						},
					},
				},
			),
			sbomPaths: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom",
			},
			sbomNames: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom",
			},
		},
		{
			desc: "catalog binary artifacts with env in arguments",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "binary",
							Args: []string{
								"--file",
								"$document",
								"--output",
								"spdx-json",
								"$artifact",
							},
							Documents: []string{
								"{{ .ArtifactName }}.{{ .Env.TEST_USER }}.sbom",
							},
						},
					},
					Env: []string{
						"TEST_USER=test-user-name",
					},
				},
			),
			sbomPaths: []string{
				"artifact3-name.test-user-name.sbom",
				"artifact4.test-user-name.sbom",
			},
			sbomNames: []string{
				"artifact3-name.test-user-name.sbom",
				"artifact4.test-user-name.sbom",
			},
		},
		{
			desc: "cataloging 'any' artifacts fails",
			ctx: context.New(
				config.Project{
					SBOMs: []config.SBOM{
						{
							Artifacts: "any",
							Cmd:       "false",
						},
					},
				},
			),
			expectedErrMsg: "cataloging artifacts: false failed: exit status 1: ",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testSBOMCataloging(t, test.ctx, test.sbomPaths, test.sbomNames, test.expectedErrMsg)
		})
	}
}

func testSBOMCataloging(tb testing.TB, ctx *context.Context, sbomPaths, sbomNames []string, expectedErrMsg string) {
	tb.Helper()
	tmpdir := tb.TempDir()

	ctx.Config.Dist = tmpdir
	ctx.Version = "1.2.2"

	// create some fake artifacts
	artifacts := []string{"artifact1", "artifact2", "artifact3", "package1.deb"}
	require.NoError(tb, os.Mkdir(filepath.Join(tmpdir, "linux_amd64"), os.ModePerm))
	for _, f := range artifacts {
		file := filepath.Join(tmpdir, f)
		require.NoError(tb, os.WriteFile(file, []byte("foo"), 0o644))
	}
	require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "linux_amd64", "artifact4"), []byte("foo"), 0o644))
	artifacts = append(artifacts, "linux_amd64/artifact4")
	require.NoError(tb, os.WriteFile(filepath.Join(tmpdir, "artifact5.tar.gz"), []byte("foo"), 0o644))
	artifacts = append(artifacts, "artifact5.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact1",
		Path: filepath.Join(tmpdir, "artifact1"),
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact2",
		Path: filepath.Join(tmpdir, "artifact2"),
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo3",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "artifact3-name",
		Path:   filepath.Join(tmpdir, "artifact3"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraBinary: "artifact3-name",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "artifact4",
		Path:   filepath.Join(tmpdir, "linux_amd64", "artifact4"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo3",
			artifact.ExtraBinary: "artifact4-name",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact5.tar.gz",
		Path: filepath.Join(tmpdir, "artifact5.tar.gz"),
		Type: artifact.UploadableSourceArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "package1.deb",
		Path: filepath.Join(tmpdir, "package1.deb"),
		Type: artifact.LinuxPackage,
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})

	// configure the pipeline
	require.NoError(tb, Pipe{}.Default(ctx))

	// run the pipeline
	if expectedErrMsg != "" {
		err := Pipe{}.Run(ctx)
		require.Error(tb, err)
		require.Contains(tb, err.Error(), expectedErrMsg)
		return
	}

	require.NoError(tb, Pipe{}.Run(ctx))

	// ensure all artifacts have an ID
	for _, arti := range ctx.Artifacts.Filter(artifact.ByType(artifact.SBOM)).List() {
		require.NotEmptyf(tb, arti.ID(), ".Extra.ID on %s", arti.Path)
	}

	// verify that only the artifacts and the sboms are in the dist dir
	gotFiles := []string{}

	require.NoError(tb, filepath.Walk(tmpdir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(tmpdir, path)
			if err != nil {
				return err
			}
			gotFiles = append(gotFiles, relPath)
			return nil
		}),
	)

	wantFiles := append(artifacts, sbomPaths...)
	sort.Strings(wantFiles)
	require.ElementsMatch(tb, wantFiles, gotFiles, "SBOM paths differ")

	var sbomArtifacts []string
	for _, sig := range ctx.Artifacts.Filter(artifact.ByType(artifact.SBOM)).List() {
		sbomArtifacts = append(sbomArtifacts, sig.Name)
	}

	require.ElementsMatch(tb, sbomArtifacts, sbomNames, "SBOM names differ")
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
			assert.Equal(t, test.expects, actual)
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
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-it",
			"Binary":         "binary-name",
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

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
			dist:     "/somewhere/to/dist",
			expectedPaths: []string{
				"/somewhere/to/dist/name-it.sbom",
			},
			expectedValues: map[string]string{
				"artifact":   "to/a/place",
				"artifactID": "id-it",
				"document":   "/somewhere/to/dist/name-it.sbom",
				"document0":  "/somewhere/to/dist/name-it.sbom",
			},
		},
		{
			name:     "default configuration + relative dist",
			artifact: art,
			cfg:      config.SBOM{},
			dist:     "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/name-it.sbom"),
			},
			expectedValues: map[string]string{
				"artifact":   "to/a/place", // note: this is always relative to ${dist}
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/name-it.sbom"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/name-it.sbom"),
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
					"${artifact}.cdx.sbom",
				},
			},
			dist: "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom"),
			},
			expectedValues: map[string]string{
				"artifact":   "to/a/place",
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/to/a/place.cdx.sbom"),
			},
		},
		{
			name:     "custom document using build vars",
			artifact: art,
			cfg: config.SBOM{
				Documents: []string{
					"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.cdx.sbom",
				},
			},
			version: "1.0.0",
			dist:    "somewhere/to/dist",
			expectedPaths: []string{
				filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
			},
			expectedValues: map[string]string{
				"artifact":   "to/a/place",
				"artifactID": "id-it",
				"document":   filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
				"document0":  filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
			},
		},
		{
			name:     "env vars with go templated options",
			artifact: art,
			cfg: config.SBOM{
				Documents: []string{
					"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.cdx.sbom",
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
				filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
			},
			expectedValues: map[string]string{
				"artifact":     "to/a/place",
				"artifactID":   "id-it",
				"with-env-var": "value",
				"custom-os":    "darwin-unique",
				"custom-arch":  "amd64-unique",
				"document":     filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
				"document0":    filepath.Join(wd, "somewhere/to/dist/binary-name_1.0.0_darwin_amd64.cdx.sbom"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.New(config.Project{
				Dist: tt.dist,
			})
			ctx.Version = tt.version

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
