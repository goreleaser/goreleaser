package nfpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeNoFormats(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"))

	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeError(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: t.TempDir(),
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb"},
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: "{{.ConventionalFileName}}",
				},
			},
		},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "testdata/testfile.txt",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "foo",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "nfpm failed for _0.0.0~rc0_amd64.deb: package name must be provided")
}

func TestRunPipeInvalidFormat(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "nope",
		NFPMs: []config.NFPM{
			{
				Bindir:  "/usr/bin",
				Formats: []string{"nope"},
				Builds:  []string{"foo"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName:      "foo",
					FileNameTemplate: defaultNameTemplate,
				},
			},
		},
	}, testctx.WithVersion("1.2.3"), testctx.WithCurrentTag("v1.2.3"))

	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   "testdata/testfile.txt",
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraID: "foo",
				},
			})
		}
	}
	require.Contains(t, Pipe{}.Run(ctx).Error(), `no packager registered for the format nope`)
}

func TestSkipOne(t *testing.T) {
	t.Helper()
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	binPath := filepath.ToSlash(filepath.Join(dist, "mybin"))
	require.NoError(t, os.WriteFile(binPath, nil, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID: "this configuration will be ignored as it has no formats",
			},
			{
				Formats: []string{"deb", "rpm"},
			},
		},
	}, testctx.WithVersion("1.0.0"))

	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "subdir/mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
			})
		}
	}
	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.True(t, pipe.IsSkip(err), err)

	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 4)
	for _, pkg := range packages {
		require.NotEmpty(t, pkg.Format())
		require.Contains(t, []string{
			"mybin_1.0.0_linux_arm64.deb",
			"mybin_1.0.0_linux_amd64.deb",
			"mybin_1.0.0_linux_amd64.rpm",
			"mybin_1.0.0_linux_arm64.rpm",
		}, pkg.Name, "package name is not expected")
	}
}

func TestInvalidTemplate(t *testing.T) {
	makeCtx := func() *context.Context {
		ctx := testctx.WrapWithCfg(t.Context(),
			config.Project{
				ProjectName: "test",
				NFPMs: []config.NFPM{
					{
						Formats: []string{"deb"},
						Builds:  []string{"default"},
					},
				},
			},
			testctx.WithVersion("1.2.3"))

		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]any{
				artifact.ExtraID: "default",
			},
		})
		return ctx
	}

	t.Run("filename_template", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Meta = true
		ctx.Config.NFPMs[0].NFPMOverridables = config.NFPMOverridables{
			FileNameTemplate: "{{.Foo}",
		}
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("source", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].NFPMOverridables = config.NFPMOverridables{
			Contents: []config.NFPMContent{
				{
					Source:      "{{ .NOPE_SOURCE }}",
					Destination: "/foo",
				},
			},
		}
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("target", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].NFPMOverridables = config.NFPMOverridables{
			Contents: []config.NFPMContent{
				{
					Source:      "./testdata/testfile.txt",
					Destination: "{{ .NOPE_TARGET }}",
				},
			},
		}
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("description", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Description = "{{ .NOPE_DESC }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("maintainer", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Maintainer = "{{ .NOPE_DESC }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("homepage", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Homepage = "{{ .NOPE_HOMEPAGE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("deb key file", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Deb.Signature.KeyFile = "{{ .NOPE_KEY_FILE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("rpm key file", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].RPM.Signature.KeyFile = "{{ .NOPE_KEY_FILE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("apk key file", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].APK.Signature.KeyFile = "{{ .NOPE_KEY_FILE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("apk key name", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].APK.Signature.KeyName = "{{ .NOPE_KEY_FILE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("bindir", func(t *testing.T) {
		ctx := makeCtx()
		ctx.Config.NFPMs[0].Bindir = "/usr/{{ .NOPE }}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func TestRunPipeInvalidContentsSourceTemplate(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		NFPMs: []config.NFPM{
			{
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "{{.asdsd}",
							Destination: "testfile",
						},
					},
				},
				Formats: []string{"deb"},
				Builds:  []string{"default"},
			},
		},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestNoBuildsFound(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb"},
				IDs:     []string{"nope"},
			},
		},
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `no linux/unix binaries found for builds [nope]`)
}

func TestCreateFileDoesntExist(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "asd",
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb", "rpm", "ipk"},
				Builds:  []string{"default"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/var/lib/test/testfile.txt",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.2.3"), testctx.WithCurrentTag("v1.2.3"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})
	require.Contains(t, Pipe{}.Run(ctx).Error(), `dist/mybin/mybin": file does not exist`)
}

func TestInvalidConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: dist,
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb"},
				Builds:  []string{"default"},
			},
		},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})
	require.Contains(t, Pipe{}.Run(ctx).Error(), `package name must be provided`)
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foobar",
		NFPMs: []config.NFPM{
			{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/usr/bin", ctx.Config.NFPMs[0].Bindir)
	require.Empty(t, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, defaultNameTemplate, ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.NFPMs[0].PackageName)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		NFPMs: []config.NFPM{
			{
				Builds: []string{"foo"},
				Bindir: "/bin",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: "foo",
				},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, "foo", ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, []string{"foo"}, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, config.NFPMRPMScripts{}, ctx.Config.NFPMs[0].RPM.Scripts)
}

func TestOverrides(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		NFPMs: []config.NFPM{
			{
				Bindir: "/bin",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: "foo",
				},
				Overrides: map[string]config.NFPMOverridables{
					"deb": {
						FileNameTemplate: "bar",
					},
				},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	merged, err := mergeOverrides(ctx.Config.NFPMs[0], "deb")
	require.NoError(t, err)
	require.Equal(t, "/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, "foo", ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, "bar", ctx.Config.NFPMs[0].Overrides["deb"].FileNameTemplate)
	require.Equal(t, "bar", merged.FileNameTemplate)
}

func TestRPMSpecificScriptsConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:      "someid",
				Builds:  []string{"default"},
				Formats: []string{"rpm"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					RPM: config.NFPMRPM{
						Scripts: config.NFPMRPMScripts{
							PreTrans:  "/does/not/exist_pretrans.sh",
							PostTrans: "/does/not/exist_posttrans.sh",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})

	t.Run("PreTrans script file does not exist", func(t *testing.T) {
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})

	t.Run("PostTrans script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].RPM.Scripts.PreTrans = "testdata/testfile.txt"
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})

	t.Run("pretrans and posttrans scriptlets set", func(t *testing.T) {
		ctx.Config.NFPMs[0].RPM.Scripts.PreTrans = "testdata/testfile.txt"
		ctx.Config.NFPMs[0].RPM.Scripts.PostTrans = "testdata/testfile.txt"

		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestAPKSpecificConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:         "someid",
				Maintainer: "me@me",
				Builds:     []string{"default"},
				Formats:    []string{"apk"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
					},
					APK: config.NFPMAPK{
						Signature: config.NFPMAPKSignature{
							KeyFile: "./testdata/rsa.priv",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})

	t.Run("no passphrase set", func(t *testing.T) {
		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`key is encrypted but no passphrase was provided`,
		)
	})

	t.Run("general passphrase set", func(t *testing.T) {
		ctx.Env = map[string]string{
			"NFPM_SOMEID_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("packager specific passphrase set", func(t *testing.T) {
		ctx.Env = map[string]string{
			"NFPM_SOMEID_APK_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestAPKSpecificScriptsConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	scripts := config.NFPMAPKScripts{
		PreUpgrade:  "/does/not/exist_preupgrade.sh",
		PostUpgrade: "/does/not/exist_postupgrade.sh",
	}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:         "someid",
				Maintainer: "me@me",
				Builds:     []string{"default"},
				Formats:    []string{"apk"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
					},
					APK: config.NFPMAPK{
						Scripts: scripts,
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})

	t.Run("PreUpgrade script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts = scripts
		ctx.Config.NFPMs[0].APK.Scripts.PostUpgrade = "testdata/testfile.txt"
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})

	t.Run("PostUpgrade script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts = scripts
		ctx.Config.NFPMs[0].APK.Scripts.PreUpgrade = "testdata/testfile.txt"
		require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
	})

	t.Run("preupgrade and postupgrade scriptlets set", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts.PreUpgrade = "testdata/testfile.txt"
		ctx.Config.NFPMs[0].APK.Scripts.PostUpgrade = "testdata/testfile.txt"

		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestIPKSpecificConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:         "someid",
				Maintainer: "me@me",
				Builds:     []string{"default"},
				Formats:    []string{"ipk"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
					},
					IPK: config.NFPMIPK{
						ABIVersion: "1.0",
						Alternatives: []config.NFPMIPKAlternative{
							{
								Priority: 100,
								Target:   "/usr/bin/mybin",
								LinkName: "/usr/bin/myaltbin",
							},
						},
						AutoInstalled: true,
						Essential:     true,
						Predepends:    []string{"libc"},
						Tags:          []string{"foo", "bar"},
						Fields: map[string]string{
							"CustomField": "customValue",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})

	t.Run("everything is fine", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx))
		ipks := ctx.Artifacts.
			Filter(artifact.ByExt("ipk")).
			List()
		require.Len(t, ipks, 1)
	})
}

func TestSeveralNFPMsWithTheSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		NFPMs: []config.NFPM{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 nfpms with the ID 'a', please fix your config")
}

func TestMeta(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Bindir:      "/usr/bin",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.github.io",
				Meta:        true,
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: defaultNameTemplate + "-{{ .Release }}-{{ .Epoch }}",
					PackageName:      "foo",
					Dependencies:     []string{"make"},
					Recommends:       []string{"svn"},
					Suggests:         []string{"bzr"},
					Replaces:         []string{"fish"},
					Conflicts:        []string{"git"},
					Release:          "10",
					Epoch:            "20",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope.conf",
							Type:        "config",
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope-rpm.conf",
							Type:        "config",
							Packager:    "rpm",
						},
						{
							Destination: "/var/log/foobar",
							Type:        "dir",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraID: "default",
				},
			})
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 2)
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Equal(t, pkg.Name, "foo_1.0.0_linux_all-10-20."+format)
		require.Equal(t, "someid", pkg.ID())
		require.Equal(t, "all", pkg.Goarch)
		require.Equal(t, "linux", pkg.Goos)
		require.ElementsMatch(t, []string{
			"/var/log/foobar",
			"/usr/share/testfile.txt",
			"/etc/nope.conf",
			"/etc/nope-rpm.conf",
		}, destinations(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
	}

	require.Len(t, ctx.Config.NFPMs[0].Contents, 4, "should not modify the config file list")
}

func TestMetaNoBinaries(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "testpkg",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Bindir:      "/usr/bin",
				Formats:     []string{"deb", "rpm"},
				Description: "Some description",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.github.io",
				Meta:        true,
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
						{
							Destination: "/var/log/foobar",
							Type:        "dir",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 2)
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Equal(t, "someid", pkg.ID())
		require.Equal(t, "all", pkg.Goarch)
		require.Equal(t, "linux", pkg.Goos)
		require.ElementsMatch(t, []string{
			"/var/log/foobar",
			"/usr/share/testfile.txt",
		}, destinations(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
	}
}

func TestSkipSign(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	_, err := os.Create(binPath)
	require.NoError(t, err)
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:      "someid",
				Builds:  []string{"default"},
				Formats: []string{"deb", "rpm", "apk"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName:      "foo",
					FileNameTemplate: defaultNameTemplate,
					Contents: []config.NFPMContent{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
					},
					Deb: config.NFPMDeb{
						Signature: config.NFPMDebSignature{
							KeyFile: "/does/not/exist.gpg",
						},
					},
					RPM: config.NFPMRPM{
						Signature: config.NFPMRPMSignature{
							KeyFile: "/does/not/exist.gpg",
						},
					},
					APK: config.NFPMAPK{
						Signature: config.NFPMAPKSignature{
							KeyFile: "/does/not/exist.gpg",
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   binPath,
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.Binary,
		Extra: map[string]any{
			artifact.ExtraID: "default",
		},
	})

	t.Run("skip sign not set", func(t *testing.T) {
		// TODO: once https://github.com/goreleaser/nfpm/pull/630 is released,
		// use require.ErrorIs() here.
		require.Error(t, Pipe{}.Run(ctx))
	})

	t.Run("skip sign set", func(t *testing.T) {
		skips.Set(ctx, skips.Sign)
		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestBinDirTemplating(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Env: []string{
			"PRO=pro",
			"DESC=templates",
			"MAINTAINER=me@me",
		},
		NFPMs: []config.NFPM{
			{
				ID: "someid",

				Bindir:      "/usr/lib/{{ .Env.PRO }}/nagios/plugins",
				Builds:      []string{"default"},
				Formats:     []string{"rpm"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description with {{ .Env.DESC }}",
				License:     "MIT",
				Maintainer:  "{{ .Env.MAINTAINER }}",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.com/{{ .Env.PRO }}",
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	for _, goos := range []string{"linux"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "subdir/mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]any{
					artifact.ExtraID: "default",
				},
			})
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()

	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		// the final binary should contain the evaluated bindir (after template eval)
		require.ElementsMatch(t, []string{
			"/usr/lib/pro/nagios/plugins/subdir/mybin",
		}, destinations(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
	}
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		}, testctx.Skip(skips.NFPM))

		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		})

		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestTemplateExt(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: t.TempDir(),
		NFPMs: []config.NFPM{
			{
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: "a_{{ .ConventionalExtension }}_b",
					PackageName:      "foo",
				},
				Meta:       true,
				Maintainer: "foo@bar",
				Formats:    []string{"deb", "rpm", "termux.deb", "apk", "archlinux"},
			},
		},
	})

	require.NoError(t, Pipe{}.Run(ctx))

	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 5)
	names := make([]string, 0, 5)
	for _, p := range packages {
		names = append(names, p.Name)
	}

	require.ElementsMatch(t, []string{
		"a_.apk_b.apk",
		"a_.deb_b.deb",
		"a_.rpm_b.rpm",
		"a_.termux.deb_b.termux.deb",
		"a_.pkg.tar.zst_b.pkg.tar.zst",
	}, names)
}
