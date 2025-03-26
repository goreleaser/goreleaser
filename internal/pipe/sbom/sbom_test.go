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
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSBOMCatalogDefault(t *testing.T) {
	defaultArgs := []string{"$artifact", "--output", "spdx-json=$document"}
	defaultSboms := []string{
		"{{ .ArtifactName }}.sbom.json",
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
			sboms:    []string{"{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom.json"},
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
			testlib.CheckPath(t, "syft")
			ctx := testctx.NewWithCfg(config.Project{
				SBOMs: test.configs,
			})
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
	ctx := testctx.NewWithCfg(config.Project{
		SBOMs: []config.SBOM{{Artifacts: "foo"}},
	})
	err := Pipe{}.Run(ctx)
	require.EqualError(t, err, "invalid list of artifacts to catalog: foo")
}

func TestSeveralSBOMsWithTheSameID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("skip SBOM cataloging", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			SBOMs: []config.SBOM{
				{
					Artifacts: "all",
				},
			},
		}, testctx.Skip(skips.SBOM))
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
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
		expectedErrAs  any
		expectedErrMsg string
	}{
		{
			desc:           "catalog errors",
			expectedErrMsg: "failed",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "binary",
						Cmd:       "exit",
						Args:      []string{"1"},
					},
				},
			}),
		},
		{
			desc:          "invalid args template",
			expectedErrAs: &tmpl.Error{},
			ctx: testctx.NewWithCfg(config.Project{
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
			}),
		},
		{
			desc: "catalog source archives",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "source"},
				},
			}),
			sbomPaths: []string{"artifact5.tar.gz.sbom.json"},
			sbomNames: []string{"artifact5.tar.gz.sbom.json"},
		},
		{
			desc: "catalog archives",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "archive"},
				},
			}),
			sbomPaths: []string{"artifact1.sbom.json", "artifact2.sbom.json"},
			sbomNames: []string{"artifact1.sbom.json", "artifact2.sbom.json"},
		},
		{
			desc: "catalog linux packages",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "package"},
				},
			}),
			sbomPaths: []string{"package1.deb.sbom.json"},
			sbomNames: []string{"package1.deb.sbom.json"},
		},
		{
			desc: "catalog binaries",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "binary"},
				},
			}),
			sbomPaths: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
				"artifact4-name_1.2.2_linux_amd64.sbom.json",
			},
			sbomNames: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
				"artifact4-name_1.2.2_linux_amd64.sbom.json",
			},
		},
		{
			desc: "manual cataloging",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "any",
						Args: []string{
							"--output",
							"spdx-json=$document0",
							"artifact5.tar.gz",
						},
						Documents: []string{
							"final.sbom.json",
						},
					},
				},
			}),
			sbomPaths: []string{"final.sbom.json"},
			sbomNames: []string{"final.sbom.json"},
		},
		{
			desc: "multiple SBOM configs",
			ctx: testctx.NewWithCfg(config.Project{
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
						Documents: []string{"{{ .ArtifactName }}.{{ .Env.SBOM_SUFFIX }}.sbom.json"},
					},
				},
			}),
			sbomPaths: []string{
				"artifact1.s2-ish.sbom.json",
				"artifact2.s2-ish.sbom.json",
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
				"artifact4-name_1.2.2_linux_amd64.sbom.json",
			},
			sbomNames: []string{
				"artifact1.s2-ish.sbom.json",
				"artifact2.s2-ish.sbom.json",
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
				"artifact4-name_1.2.2_linux_amd64.sbom.json",
			},
		},
		{
			desc: "catalog artifacts with filtered by ID",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "binary",
						IDs:       []string{"foo"},
					},
				},
			}),
			sbomPaths: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
			},
			sbomNames: []string{
				"artifact3-name_1.2.2_linux_amd64.sbom.json",
			},
		},
		{
			desc: "catalog binary artifacts with env in arguments",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "binary",
						Args: []string{
							"--output",
							"spdx-json=$document",
							"$artifact",
						},
						Documents: []string{
							"{{ .ArtifactName }}.{{ .Env.TEST_USER }}.sbom.json",
						},
					},
				},
				Env: []string{
					"TEST_USER=test-user-name",
				},
			}),
			sbomPaths: []string{
				"artifact3-name.test-user-name.sbom.json",
				"artifact4.test-user-name.sbom.json",
			},
			sbomNames: []string{
				"artifact3-name.test-user-name.sbom.json",
				"artifact4.test-user-name.sbom.json",
			},
		},
		{
			desc: "cataloging 'any' artifacts fails",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "any",
						Cmd:       "false",
					},
				},
			}),
			expectedErrMsg: "cataloging artifacts: false failed: ",
		},
		{
			desc: "catalog wrong command",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{Args: []string{"$artifact", "--file", "$sbom", "--output", "spdx-json"}},
				},
			}),
			expectedErrMsg: "cataloging artifacts: command did not write any files, check your configuration",
		},
		{
			desc: "no matches",
			ctx: testctx.NewWithCfg(config.Project{
				SBOMs: []config.SBOM{
					{IDs: []string{"nopenopenope"}},
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testSBOMCataloging(
				t,
				test.ctx,
				test.sbomPaths,
				test.sbomNames,
				test.expectedErrAs,
				test.expectedErrMsg,
			)
		})
	}
}

func testSBOMCataloging(
	tb testing.TB,
	ctx *context.Context,
	sbomPaths, sbomNames []string,
	expectedErrAs any,
	expectedErrMsg string,
) {
	tb.Helper()
	testlib.CheckPath(tb, "syft")
	tmpdir := testlib.Mktmp(tb)

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
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "artifact2",
		Path: filepath.Join(tmpdir, "artifact2"),
		Type: artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID: "foo3",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "artifact3-name",
		Path:   filepath.Join(tmpdir, "artifact3"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableBinary,
		Extra: map[string]any{
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
		Extra: map[string]any{
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
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})

	// configure the pipeline
	require.NoError(tb, Pipe{}.Default(ctx))

	// run the pipeline
	if expectedErrMsg != "" {
		err := Pipe{}.Run(ctx)
		require.ErrorContains(tb, err, expectedErrMsg)
		return
	}
	if expectedErrAs != nil {
		require.ErrorAs(tb, Pipe{}.Run(ctx), expectedErrAs)
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
			relPath, err := filepath.Rel(filepath.FromSlash(tmpdir), filepath.FromSlash(path))
			if err != nil {
				return err
			}
			gotFiles = append(gotFiles, filepath.ToSlash(relPath))
			return nil
		}),
	)

	wantFiles := append(artifacts, sbomPaths...)
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
			ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
		SBOMs: []config.SBOM{
			{Cmd: "syft"},
			{Cmd: "foobar"},
		},
	})
	require.Equal(t, []string{"syft", "foobar"}, Pipe{}.Dependencies(ctx))
}
