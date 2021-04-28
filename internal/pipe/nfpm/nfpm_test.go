package nfpm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeNoFormats(t *testing.T) {
	ctx := &context.Context{
		Version: "1.0.0",
		Git: context.GitInfo{
			CurrentTag: "v1.0.0",
		},
		Config: config.Project{
			NFPMs: []config.NFPM{
				{},
			},
		},
		Parallelism: runtime.NumCPU(),
	}
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
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
					"ID": "foo",
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
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Bindir:      "/usr/bin",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm", "apk"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.github.io",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: defaultNameTemplate + "-{{ .Release }}-{{ .Epoch }}",
					PackageName:      "foo",
					Dependencies:     []string{"make"},
					Recommends:       []string{"svn"},
					Suggests:         []string{"bzr"},
					Replaces:         []string{"fish"},
					Conflicts:        []string{"git"},
					EmptyFolders:     []string{"/var/log/foobar"},
					Release:          "10",
					Epoch:            "20",
					Contents: []*files.Content{
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
					"ID": "default",
				},
			})
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 6)
	for _, pkg := range packages {
		format := pkg.ExtraOr("Format", "").(string)
		require.NotEmpty(t, format)
		require.Equal(t, pkg.Name, "foo_1.0.0_Tux_"+pkg.Goarch+"-10-20."+format)
		require.Equal(t, pkg.ExtraOr("ID", ""), "someid")
		require.ElementsMatch(t, []string{
			"/usr/share/testfile.txt",
			"/etc/nope.conf",
			"/etc/nope-rpm.conf",
			"/etc/nope2.conf",
			"/usr/bin/mybin",
		}, destinations(pkg.ExtraOr("Files", files.Contents{}).(files.Contents)))
	}
	require.Len(t, ctx.Config.NFPMs[0].Contents, 4, "should not modify the config file list")
}

func TestInvalidNameTemplate(t *testing.T) {
	ctx := &context.Context{
		Parallelism: runtime.NumCPU(),
		Artifacts:   artifact.New(),
		Config: config.Project{
			NFPMs: []config.NFPM{
				{
					NFPMOverridables: config.NFPMOverridables{
						PackageName:      "foo",
						FileNameTemplate: "{{.Foo}",
					},
					Formats: []string{"deb"},
					Builds:  []string{"default"},
				},
			},
		},
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"ID": "default",
		},
	})
	require.Contains(t, Pipe{}.Run(ctx).Error(), `template: tmpl:1: unexpected "}" in operand`)
}

func TestNoBuildsFound(t *testing.T) {
	ctx := &context.Context{
		Parallelism: runtime.NumCPU(),
		Artifacts:   artifact.New(),
		Config: config.Project{
			NFPMs: []config.NFPM{
				{
					Formats: []string{"deb"},
					Builds:  []string{"nope"},
				},
			},
		},
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"ID": "default",
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
			"ID": "default",
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
	ctx.Version = "v1.2.3"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   filepath.Join(dist, "mybin", "mybin"),
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"ID": "default",
		},
	})
	require.Contains(t, Pipe{}.Run(ctx).Error(), `invalid nfpm config: package name must be provided`)
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			ProjectName: "foobar",
			NFPMs: []config.NFPM{
				{},
			},
			Builds: []config.Build{
				{ID: "foo"},
				{ID: "bar"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/usr/local/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, []string{"foo", "bar"}, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, defaultNameTemplate, ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.NFPMs[0].PackageName)
}

func TestDefaultDeprecatedOptions(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			ProjectName: "foobar",
			NFPMs: []config.NFPM{
				{
					NFPMOverridables: config.NFPMOverridables{
						Files: map[string]string{
							"testdata/testfile.txt": "/bin/foo",
						},
						ConfigFiles: map[string]string{
							"testdata/testfile.txt": "/etc/foo.conf",
						},
						Symlinks: map[string]string{
							"/etc/foo.conf": "/etc/foov2.conf",
						},
						RPM: config.NFPMRPM{
							GhostFiles: []string{"/etc/ghost.conf"},
							ConfigNoReplaceFiles: map[string]string{
								"testdata/testfile.txt": "/etc/foo_keep.conf",
							},
						},
						Deb: config.NFPMDeb{
							VersionMetadata: "beta1",
						},
					},
				},
			},
			Builds: []config.Build{
				{ID: "foo"},
				{ID: "bar"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/usr/local/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, []string{"foo", "bar"}, ctx.Config.NFPMs[0].Builds)
	require.ElementsMatch(t, []*files.Content{
		{Source: "testdata/testfile.txt", Destination: "/bin/foo"},
		{Source: "testdata/testfile.txt", Destination: "/etc/foo.conf", Type: "config"},
		{Source: "/etc/foo.conf", Destination: "/etc/foov2.conf", Type: "symlink"},
		{Destination: "/etc/ghost.conf", Type: "ghost", Packager: "rpm"},
		{Source: "testdata/testfile.txt", Destination: "/etc/foo_keep.conf", Type: "config|noreplace", Packager: "rpm"},
	}, ctx.Config.NFPMs[0].Contents)
	require.Equal(t, defaultNameTemplate, ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, ctx.Config.ProjectName, ctx.Config.NFPMs[0].PackageName)
	require.Equal(t, "beta1", ctx.Config.NFPMs[0].VersionMetadata)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{ID: "foo"},
				{ID: "bar"},
			},
			NFPMs: []config.NFPM{
				{
					Builds: []string{"foo"},
					Bindir: "/bin",
					NFPMOverridables: config.NFPMOverridables{
						FileNameTemplate: "foo",
					},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, "foo", ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, []string{"foo"}, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, config.NFPMRPMScripts{}, ctx.Config.NFPMs[0].RPM.Scripts)
}

func TestOverrides(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
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
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	merged, err := mergeOverrides(ctx.Config.NFPMs[0], "deb")
	require.NoError(t, err)
	require.Equal(t, "/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, "foo", ctx.Config.NFPMs[0].FileNameTemplate)
	require.Equal(t, "bar", ctx.Config.NFPMs[0].Overrides["deb"].FileNameTemplate)
	require.Equal(t, "bar", merged.FileNameTemplate)
}

func TestDebSpecificConfig(t *testing.T) {
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
				Formats: []string{"deb"},
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
					"ID": "default",
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
			"NFPM_SOMEID_DEB_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
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
					"ID": "default",
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
					"ID": "default",
				},
			})
		}
	}

	t.Run("PreTrans script file does not exist", func(t *testing.T) {
		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`nfpm failed: open /does/not/exist_pretrans.sh: no such file or directory`,
		)
	})

	t.Run("PostTrans script file does not exist", func(t *testing.T) {
		ctx.Config.NFPMs[0].RPM.Scripts.PreTrans = "testdata/testfile.txt"

		require.Contains(
			t,
			Pipe{}.Run(ctx).Error(),
			`nfpm failed: open /does/not/exist_posttrans.sh: no such file or directory`,
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
					"ID": "default",
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

func TestSeveralNFPMsWithTheSameID(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			NFPMs: []config.NFPM{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
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
					EmptyFolders:     []string{"/var/log/foobar"},
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
					"ID": "default",
				},
			})
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 4)
	for _, pkg := range packages {
		format := pkg.ExtraOr("Format", "").(string)
		require.NotEmpty(t, format)
		require.Equal(t, pkg.Name, "foo_1.0.0_Tux_"+pkg.Goarch+"-10-20."+format)
		require.Equal(t, pkg.ExtraOr("ID", ""), "someid")
		require.ElementsMatch(t, []string{
			"/usr/share/testfile.txt",
			"/etc/nope.conf",
			"/etc/nope-rpm.conf",
		}, destinations(pkg.ExtraOr("Files", files.Contents{}).(files.Contents)))
	}

	require.Len(t, ctx.Config.NFPMs[0].Contents, 3, "should not modify the config file list")
}

func TestSkipSign(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
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
					"ID": "default",
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
