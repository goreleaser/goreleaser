package nfpm

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeNoFormats(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		},
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithVersion("1.0.1"),
	)
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestDefaultsDeprecated(t *testing.T) {
	t.Run("replacements", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			NFPMs: []config.NFPM{
				{
					NFPMOverridables: config.NFPMOverridables{
						Replacements: map[string]string{
							"linux": "Tux",
						},
					},
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.True(t, ctx.Deprecated)
	})

	t.Run("replacements overrides", func(t *testing.T) {
		ctx := context.New(config.Project{
			NFPMs: []config.NFPM{
				{
					Overrides: map[string]config.NFPMOverridables{
						"apk": {
							Replacements: map[string]string{
								"linux": "Tux",
							},
						},
					},
				},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.True(t, ctx.Deprecated)
	})
}

func TestRunPipeError(t *testing.T) {
	ctx := context.New(config.Project{
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
		Extra: map[string]interface{}{
			artifact.ExtraID: "foo",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "nfpm failed for _0.0.0~rc0_amd64.deb: package name must be provided")
}

func TestRunPipeInvalidFormat(t *testing.T) {
	ctx := context.New(config.Project{
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
	})
	ctx.Version = "1.2.3"
	ctx.Git = context.GitInfo{
		CurrentTag: "v1.2.3",
	}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   "testdata/testfile.txt",
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "foo",
				},
			})
		}
	}
	require.Contains(t, Pipe{}.Run(ctx).Error(), `no packager registered for the format nope`)
}

func TestRunPipe(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Env: []string{
			"PRO=pro",
			"DESC=templates",
		},
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Bindir:      "/usr/bin",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm", "apk", "termux.deb", "archlinux"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description with {{ .Env.DESC }}",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.com/{{ .Env.PRO }}",
				Changelog:   "./testdata/changelog.yaml",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: defaultNameTemplate + "-{{ .Release }}-{{ .Epoch }}",
					PackageName:      "foo",
					Dependencies:     []string{"make"},
					Recommends:       []string{"svn"},
					Suggests:         []string{"bzr"},
					Replaces:         []string{"fish"},
					Conflicts:        []string{"git"},
					Provides:         []string{"ash"},
					Release:          "10",
					Epoch:            "20",
					Contents: []*files.Content{
						{
							Destination: "/var/log/foobar",
							Type:        "dir",
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope.conf",
							Type:        "config",
						},
						{
							Destination: "/etc/mydir",
							Type:        "dir",
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope-rpm.conf",
							Type:        "config",
							Packager:    "rpm",
						},
						{
							Source:      "/etc/nope.conf",
							Destination: "/etc/nope2.conf",
							Type:        "symlink",
						},
						{
							Source:      "./testdata/testfile-{{ .Arch }}{{.Amd64}}{{.Arm}}{{.Mips}}.txt",
							Destination: "/etc/nope3_{{ .ProjectName }}.conf",
						},
						{
							Source:      "./testdata/folder",
							Destination: "/etc/folder",
						},
					},
					Replacements: map[string]string{
						"linux": "Tux",
					},
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin", "ios"} {
		for _, goarch := range []string{"amd64", "386", "arm64", "arm", "mips"} {
			if goos == "ios" && goarch != "arm64" {
				continue
			}
			switch goarch {
			case "arm":
				for _, goarm := range []string{"6", "7"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "subdir/mybin",
						Path:   binPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			case "amd64":
				for _, goamd64 := range []string{"v1", "v2", "v3", "v4"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "subdir/mybin",
						Path:    binPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			case "mips":
				for _, gomips := range []string{"softfloat", "hardfloat"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "subdir/mybin",
						Path:   binPath,
						Goarch: goarch,
						Goos:   goos,
						Gomips: gomips,
						Type:   artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			default:
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "subdir/mybin",
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Binary,
					Extra: map[string]interface{}{
						artifact.ExtraID: "default",
					},
				})
			}
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 47)
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		arch := pkg.Goarch
		if pkg.Goarm != "" {
			arch += "v" + pkg.Goarm
		}
		if pkg.Goamd64 != "v1" {
			arch += pkg.Goamd64
		}
		if pkg.Gomips != "" {
			arch += "_" + pkg.Gomips
		}

		ext := "." + format
		if format != "termux.deb" {
			packager, err := nfpm.Get(format)
			require.NoError(t, err)

			if packager, ok := packager.(nfpm.PackagerWithExtension); ok {
				ext = packager.ConventionalExtension()
			}
		}

		if pkg.Goos == "linux" {
			require.Equal(t, "foo_1.0.0_Tux_"+arch+"-10-20"+ext, pkg.Name)
		} else {
			require.Equal(t, "foo_1.0.0_ios_arm64-10-20"+ext, pkg.Name)
		}
		require.Equal(t, "someid", pkg.ID())
		require.ElementsMatch(t, []string{
			"./testdata/testfile.txt",
			"./testdata/testfile.txt",
			"./testdata/testfile.txt",
			"/etc/nope.conf",
			"./testdata/folder",
			"./testdata/testfile-" + pkg.Goarch + pkg.Goamd64 + pkg.Goarm + pkg.Gomips + ".txt",
			binPath,
		}, sources(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))

		bin := "/usr/bin/subdir/"
		if format == termuxFormat {
			bin = filepath.Join("/data/data/com.termux/files", bin)
		}
		bin = filepath.Join(bin, "mybin")
		require.ElementsMatch(t, []string{
			"/var/log/foobar",
			"/usr/share/testfile.txt",
			"/etc/mydir",
			"/etc/nope.conf",
			"/etc/nope-rpm.conf",
			"/etc/nope2.conf",
			"/etc/nope3_mybin.conf",
			"/etc/folder",
			bin,
		}, destinations(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))
	}
	require.Len(t, ctx.Config.NFPMs[0].Contents, 8, "should not modify the config file list")
}

func TestRunPipeConventionalNameTemplate(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm", "apk", "archlinux"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description ",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.com/",
				Bindir:      "/usr/bin",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: `{{ trimsuffix (trimsuffix (trimsuffix (trimsuffix .ConventionalFileName ".pkg.tar.zst") ".deb") ".rpm") ".apk" }}{{ if not (eq .Amd64 "v1")}}{{ .Amd64 }}{{ end }}`,
					PackageName:      "foo",
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm64", "arm", "mips"} {
			switch goarch {
			case "arm":
				for _, goarm := range []string{"6", "7"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "subdir/mybin",
						Path:   binPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			case "amd64":
				for _, goamd64 := range []string{"v1", "v2", "v3", "v4"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "subdir/mybin",
						Path:    binPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			case "mips":
				for _, gomips := range []string{"softfloat", "hardfloat"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "subdir/mybin",
						Path:   binPath,
						Goarch: goarch,
						Goos:   goos,
						Gomips: gomips,
						Type:   artifact.Binary,
						Extra: map[string]interface{}{
							artifact.ExtraID: "default",
						},
					})
				}
			default:
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "subdir/mybin",
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Binary,
					Extra: map[string]interface{}{
						artifact.ExtraID: "default",
					},
				})
			}
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 40)
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Contains(t, []string{
			"foo-1.0.0.aarch64.rpm",
			"foo-1.0.0.armv6hl.rpm",
			"foo-1.0.0.armv7hl.rpm",
			"foo-1.0.0.i386.rpm",
			"foo-1.0.0.mipshardfloat.rpm",
			"foo-1.0.0.mipssoftfloat.rpm",
			"foo-1.0.0.x86_64.rpm",
			"foo-1.0.0.x86_64v2.rpm",
			"foo-1.0.0.x86_64v3.rpm",
			"foo-1.0.0.x86_64v4.rpm",
			"foo_1.0.0_aarch64.apk",
			"foo_1.0.0_amd64.deb",
			"foo_1.0.0_amd64v2.deb",
			"foo_1.0.0_amd64v3.deb",
			"foo_1.0.0_amd64v4.deb",
			"foo_1.0.0_arm64.deb",
			"foo_1.0.0_armhf.apk",
			"foo_1.0.0_armhf.deb",
			"foo_1.0.0_armv7.apk",
			"foo_1.0.0_i386.deb",
			"foo_1.0.0_mipshardfloat.apk",
			"foo_1.0.0_mipshardfloat.deb",
			"foo_1.0.0_mipssoftfloat.apk",
			"foo_1.0.0_mipssoftfloat.deb",
			"foo_1.0.0_x86.apk",
			"foo_1.0.0_x86_64.apk",
			"foo_1.0.0_x86_64v2.apk",
			"foo_1.0.0_x86_64v3.apk",
			"foo_1.0.0_x86_64v4.apk",
			"foo-1.0.0-1-aarch64.pkg.tar.zst",
			"foo-1.0.0-1-armv6h.pkg.tar.zst",
			"foo-1.0.0-1-armv7h.pkg.tar.zst",
			"foo-1.0.0-1-i686.pkg.tar.zst",
			"foo-1.0.0-1-x86_64.pkg.tar.zst",
			"foo-1.0.0-1-x86_64v2.pkg.tar.zst",
			"foo-1.0.0-1-x86_64v3.pkg.tar.zst",
			"foo-1.0.0-1-x86_64v4.pkg.tar.zst",
			"foo-1.0.0-1-mipssoftfloat.pkg.tar.zst",
			"foo-1.0.0-1-mipshardfloat.pkg.tar.zst",
		}, pkg.Name, "package name is not expected")
		require.Equal(t, "someid", pkg.ID())
		require.ElementsMatch(t, []string{binPath}, sources(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))
		require.ElementsMatch(t, []string{"/usr/bin/subdir/mybin"}, destinations(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))
	}
}

func TestInvalidTemplate(t *testing.T) {
	makeCtx := func() *context.Context {
		ctx := testctx.NewWithCfg(
			config.Project{
				ProjectName: "test",
				NFPMs: []config.NFPM{
					{
						Formats: []string{"deb"},
						Builds:  []string{"default"},
					},
				},
			},
			testctx.WithVersion("1.2.3"),
		)
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
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
			Contents: files.Contents{
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
			Contents: files.Contents{
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
	ctx := testctx.NewWithCfg(config.Project{
		NFPMs: []config.NFPM{
			{
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []*files.Content{
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
		Extra: map[string]interface{}{
			artifact.ExtraID: "default",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestNoBuildsFound(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb"},
				Builds:  []string{"nope"},
			},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `no linux binaries found for builds [nope]`)
}

func TestCreateFileDoesntExist(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	ctx := context.New(config.Project{
		Dist:        dist,
		ProjectName: "asd",
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb", "rpm"},
				Builds:  []string{"default"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []*files.Content{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/var/lib/test/testfile.txt",
						},
					},
				},
			},
		},
	})
	ctx.Version = "1.2.3"
	ctx.Git = context.GitInfo{
		CurrentTag: "v1.2.3",
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
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
	ctx := context.New(config.Project{
		Dist: dist,
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb"},
				Builds:  []string{"default"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "default",
		},
	})
	require.Contains(t, Pipe{}.Run(ctx).Error(), `package name must be provided`)
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := testctx.NewWithCfg(config.Project{
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

func TestDebSpecificConfig(t *testing.T) {
	setupContext := func(tb testing.TB) *context.Context {
		tb.Helper()
		folder := t.TempDir()
		dist := filepath.Join(folder, "dist")
		require.NoError(t, os.Mkdir(dist, 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
		binPath := filepath.Join(dist, "mybin", "mybin")
		f, err := os.Create(binPath)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		ctx := context.New(config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			NFPMs: []config.NFPM{
				{
					ID:         "someid",
					Builds:     []string{"default"},
					Formats:    []string{"deb"},
					Maintainer: "foo",
					NFPMOverridables: config.NFPMOverridables{
						PackageName: "foo",
						Contents: []*files.Content{
							{
								Source:      "testdata/testfile.txt",
								Destination: "/usr/share/testfile.txt",
							},
						},
						Deb: config.NFPMDeb{
							Signature: config.NFPMDebSignature{
								KeyFile: "./testdata/privkey.gpg",
							},
						},
					},
				},
			},
		})
		ctx.Version = "1.0.0"
		ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
		for _, goos := range []string{"linux", "darwin"} {
			for _, goarch := range []string{"amd64", "386"} {
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "mybin",
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Binary,
					Extra: map[string]interface{}{
						artifact.ExtraID: "default",
					},
				})
			}
		}
		return ctx
	}

	t.Run("no passphrase set", func(t *testing.T) {
		require.Contains(
			t,
			Pipe{}.Run(setupContext(t)).Error(),
			`key is encrypted but no passphrase was provided`,
		)
	})

	t.Run("general passphrase set", func(t *testing.T) {
		ctx := setupContext(t)
		ctx.Env = map[string]string{
			"NFPM_SOMEID_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("packager specific passphrase set", func(t *testing.T) {
		ctx := setupContext(t)
		ctx.Env = map[string]string{
			"NFPM_SOMEID_DEB_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("lintian", func(t *testing.T) {
		ctx := setupContext(t)
		ctx.Env = map[string]string{
			"NFPM_SOMEID_DEB_PASSPHRASE": "hunter2",
		}
		ctx.Config.NFPMs[0].NFPMOverridables.Deb.Lintian = []string{
			"statically-linked-binary",
			"changelog-file-missing-in-native-package",
		}
		require.NoError(t, Pipe{}.Run(ctx))

		for _, goarch := range []string{"amd64", "386"} {
			bts, err := os.ReadFile(filepath.Join(ctx.Config.Dist, "deb/foo_"+goarch+"/.lintian"))
			require.NoError(t, err)
			require.Equal(t, "foo: statically-linked-binary\nfoo: changelog-file-missing-in-native-package", string(bts))
		}
	})
}

func TestRPMSpecificConfig(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	f, err := os.Create(binPath)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:      "someid",
				Builds:  []string{"default"},
				Formats: []string{"rpm"},
				NFPMOverridables: config.NFPMOverridables{
					PackageName: "foo",
					Contents: []*files.Content{
						{
							Source:      "testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
						},
					},
					RPM: config.NFPMRPM{
						Signature: config.NFPMRPMSignature{
							KeyFile: "./testdata/privkey.gpg",
						},
					},
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}

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
			"NFPM_SOMEID_RPM_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
	})
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
	ctx := context.New(config.Project{
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
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}

	t.Run("PreTrans script file does not exist", func(t *testing.T) {
		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`open /does/not/exist_pretrans.sh: no such file or directory`,
		)
	})

	t.Run("PostTrans script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].RPM.Scripts.PreTrans = "testdata/testfile.txt"

		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`open /does/not/exist_posttrans.sh: no such file or directory`,
		)
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
	ctx := context.New(config.Project{
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
					Contents: []*files.Content{
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
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}

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
	ctx := context.New(config.Project{
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
					Contents: []*files.Content{
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
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}

	t.Run("PreUpgrade script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts = scripts
		ctx.Config.NFPMs[0].APK.Scripts.PostUpgrade = "testdata/testfile.txt"

		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`stat /does/not/exist_preupgrade.sh: no such file or directory`,
		)
	})

	t.Run("PostUpgrade script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts = scripts
		ctx.Config.NFPMs[0].APK.Scripts.PreUpgrade = "testdata/testfile.txt"

		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`stat /does/not/exist_postupgrade.sh: no such file or directory`,
		)
	})

	t.Run("preupgrade and postupgrade scriptlets set", func(t *testing.T) {
		ctx.Config.NFPMs[0].APK.Scripts.PreUpgrade = "testdata/testfile.txt"
		ctx.Config.NFPMs[0].APK.Scripts.PostUpgrade = "testdata/testfile.txt"

		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestSeveralNFPMsWithTheSameID(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	ctx := context.New(config.Project{
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
					Contents: []*files.Content{
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
					Replacements: map[string]string{
						"linux": "Tux",
					},
				},
			},
		},
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 4)
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Equal(t, pkg.Name, "foo_1.0.0_Tux_"+pkg.Goarch+"-10-20."+format)
		require.Equal(t, pkg.ID(), "someid")
		require.ElementsMatch(t, []string{
			"/var/log/foobar",
			"/usr/share/testfile.txt",
			"/etc/nope.conf",
			"/etc/nope-rpm.conf",
		}, destinations(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))
	}

	require.Len(t, ctx.Config.NFPMs[0].Contents, 4, "should not modify the config file list")
}

func TestSkipSign(t *testing.T) {
	folder, err := os.MkdirTemp("", "archivetest")
	require.NoError(t, err)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.Join(dist, "mybin", "mybin")
	_, err = os.Create(binPath)
	require.NoError(t, err)
	ctx := context.New(config.Project{
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
					Contents: []*files.Content{
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
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: "default",
				},
			})
		}
	}

	t.Run("skip sign not set", func(t *testing.T) {
		contains := "open /does/not/exist.gpg: no such file or directory"
		if runtime.GOOS == "windows" {
			contains = "open /does/not/exist.gpg: The system cannot find the path specified."
		}
		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			contains,
		)
	})

	t.Run("skip sign set", func(t *testing.T) {
		ctx.SkipSign = true
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
	ctx := context.New(config.Project{
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
				// Bindir should pass through the template engine
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
	})
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.0"}
	for _, goos := range []string{"linux"} {
		for _, goarch := range []string{"amd64", "386"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "subdir/mybin",
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
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
		}, destinations(artifact.ExtraOr(*pkg, extraFiles, files.Contents{})))
	}
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func sources(contents files.Contents) []string {
	result := make([]string, 0, len(contents))
	for _, f := range contents {
		if f.Source == "" {
			continue
		}
		result = append(result, f.Source)
	}
	return result
}
