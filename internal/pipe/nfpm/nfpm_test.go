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
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeNoFormats(t *testing.T) {
	var ctx = &context.Context{
		Version: "1.0.0",
		Git: context.GitInfo{
			CurrentTag: "v1.0.0",
		},
		Config:      config.Project{},
		Parallelism: runtime.NumCPU(),
	}
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidFormat(t *testing.T) {
	var ctx = context.New(config.Project{
		ProjectName: "nope",
		NFPMs: []config.NFPM{
			{
				Bindir:  "/usr/bin",
				Formats: []string{"nope"},
				Builds:  []string{"foo"},
				NFPMOverridables: config.NFPMOverridables{
					NameTemplate: defaultNameTemplate,
					Files:        map[string]string{},
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
				Path:   "whatever",
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
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var binPath = filepath.Join(dist, "mybin", "mybin")
	_, err = os.Create(binPath)
	require.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				Bindir:      "/usr/bin",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm"},
				Description: "Some description",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.github.io",
				NFPMOverridables: config.NFPMOverridables{
					NameTemplate: defaultNameTemplate,
					Dependencies: []string{"make"},
					Recommends:   []string{"svn"},
					Suggests:     []string{"bzr"},
					Conflicts:    []string{"git"},
					EmptyFolders: []string{"/var/log/foobar"},
					Files: map[string]string{
						"./testdata/testfile.txt": "/usr/share/testfile.txt",
					},
					ConfigFiles: map[string]string{
						"./testdata/testfile.txt": "/etc/nope.conf",
					},
					Replacements: map[string]string{
						"linux": "Tux",
					},
				},
				Overrides: map[string]config.NFPMOverridables{
					"rpm": {
						ConfigFiles: map[string]string{
							"./testdata/testfile.txt": "/etc/nope-rpm.conf",
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
	require.NoError(t, Pipe{}.Run(ctx))
	var packages = ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 4)
	for _, pkg := range packages {
		require.Contains(t, pkg.Name, "mybin_1.0.0_Tux_", "linux should have been replaced by Tux")
	}
	require.Len(t, ctx.Config.NFPMs[0].Files, 1, "should not modify the config file list")
}

func TestInvalidNameTemplate(t *testing.T) {
	var ctx = &context.Context{
		Parallelism: runtime.NumCPU(),
		Artifacts:   artifact.New(),
		Config: config.Project{
			NFPMs: []config.NFPM{
				{
					NFPMOverridables: config.NFPMOverridables{NameTemplate: "{{.Foo}"},
					Formats:          []string{"deb"},
					Builds:           []string{"default"},
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
	var ctx = &context.Context{
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
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var ctx = context.New(config.Project{
		Dist:        dist,
		ProjectName: "asd",
		NFPMs: []config.NFPM{
			{
				Formats: []string{"deb", "rpm"},
				Builds:  []string{"default"},
				NFPMOverridables: config.NFPMOverridables{
					Files: map[string]string{
						"testdata/testfile.txt": "/var/lib/test/testfile.txt",
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
	require.Contains(t, Pipe{}.Run(ctx).Error(), `dist/mybin/mybin: file does not exist`)
}

func TestInvalidConfig(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
	var ctx = context.New(config.Project{
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
	require.Contains(t, Pipe{}.Run(ctx).Error(), `invalid nfpm config: package name cannot be empty`)
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			NFPMs: []config.NFPM{},
			Builds: []config.Build{
				{ID: "foo"},
				{ID: "bar"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/usr/local/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, []string{"foo", "bar"}, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, defaultNameTemplate, ctx.Config.NFPMs[0].NameTemplate)
}

func TestDefaultDeprecate(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			NFPM: config.NFPM{
				Formats: []string{"deb"},
			},
			Builds: []config.Build{
				{ID: "foo"},
				{ID: "bar"},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/usr/local/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, []string{"deb"}, ctx.Config.NFPMs[0].Formats)
	require.Equal(t, []string{"foo", "bar"}, ctx.Config.NFPMs[0].Builds)
	require.Equal(t, defaultNameTemplate, ctx.Config.NFPMs[0].NameTemplate)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
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
						NameTemplate: "foo",
					},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "/bin", ctx.Config.NFPMs[0].Bindir)
	require.Equal(t, "foo", ctx.Config.NFPMs[0].NameTemplate)
	require.Equal(t, []string{"foo"}, ctx.Config.NFPMs[0].Builds)
}

func TestOverrides(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			NFPMs: []config.NFPM{
				{
					Bindir: "/bin",
					NFPMOverridables: config.NFPMOverridables{
						NameTemplate: "foo",
					},
					Overrides: map[string]config.NFPMOverridables{
						"deb": {
							NameTemplate: "bar",
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
	require.Equal(t, "foo", ctx.Config.NFPMs[0].NameTemplate)
	require.Equal(t, "bar", ctx.Config.NFPMs[0].Overrides["deb"].NameTemplate)
	require.Equal(t, "bar", merged.NameTemplate)
}

func TestSeveralNFPMsWithTheSameID(t *testing.T) {
	var ctx = &context.Context{
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
