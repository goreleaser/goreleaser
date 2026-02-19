//go:build integration

package sbom

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/archive"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSBOMCatalogDefault(t *testing.T) {
	testlib.CheckPath(t, "syft")
	defaultArgs := []string{"$artifact", "--output", "spdx-json=$document", "--enrich", "all"}
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
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIntegrationSBOMCatalogArtifacts(t *testing.T) {
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
			expectedErrMsg: "could not catalog artifact",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "source"},
				},
			}),

			sbomPaths: []string{"artifact5.tar.gz.sbom.json"},
			sbomNames: []string{"artifact5.tar.gz.sbom.json"},
		},
		{
			desc: "catalog archives",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "archive"},
				},
			}),

			sbomPaths: []string{"artifact1.sbom.json", "artifact2.sbom.json"},
			sbomNames: []string{"artifact1.sbom.json", "artifact2.sbom.json"},
		},
		{
			desc: "catalog linux packages",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
				SBOMs: []config.SBOM{
					{Artifacts: "package"},
				},
			}),

			sbomPaths: []string{"package1.deb.sbom.json"},
			sbomNames: []string{"package1.deb.sbom.json"},
		},
		{
			desc: "catalog binaries",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
				SBOMs: []config.SBOM{
					{
						Artifacts: "any",
						Cmd:       "false",
					},
				},
			}),

			expectedErrMsg: "could not catalog artifact",
		},
		{
			desc: "catalog wrong command",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
				SBOMs: []config.SBOM{
					{Args: []string{"$artifact", "--file", "$sbom", "--output", "spdx-json"}},
				},
			}),

			expectedErrMsg: "cataloging artifacts: command did not write any files, check your configuration",
		},
		{
			desc: "no matches",
			ctx: testctx.WrapWithCfg(t.Context(), config.Project{
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

	tgz, err := os.OpenFile(filepath.Join(tmpdir, "artifact5.tar.gz"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o644)
	require.NoError(tb, err)
	tb.Cleanup(func() { _ = tgz.Close() })
	a, err := archive.New(tgz, "tar.gz")
	require.NoError(tb, err)
	require.NoError(tb, a.Add(config.File{
		Source:      filepath.Join(tmpdir, "linux_amd64", "artifact4"),
		Destination: "artifact",
	}))
	require.NoError(tb, a.Close())

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
		require.Error(tb, err)

		de, ok := errors.AsType[gerrors.ErrDetailed](err)
		if ok {
			require.Contains(tb, de.Messages(), expectedErrMsg)
		} else {
			require.ErrorContains(tb, err, expectedErrMsg)
		}
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
