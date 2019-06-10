package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/archive"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestRunPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(tt *testing.T) {
			var dist = filepath.Join(folder, format+"_dist")
			require.NoError(t, os.Mkdir(dist, 0755))
			require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
			require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0755))
			_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
			require.NoError(t, err)
			_, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
			require.NoError(t, err)
			_, err = os.Create(filepath.Join(folder, "README.md"))
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(filepath.Join(folder, "foo", "bar", "foobar"), 0755))
			_, err = os.Create(filepath.Join(filepath.Join(folder, "foo", "bar", "foobar", "blah.txt")))
			require.NoError(t, err)
			var ctx = context.New(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							ID:           "defaultarch",
							Builds:       []string{"default"},
							NameTemplate: defaultNameTemplate,
							Files: []string{
								"README.*",
								"./foo/**/*",
							},
							FormatOverrides: []config.FormatOverride{
								{
									Goos:   "windows",
									Format: "zip",
								},
							},
						},
					},
				},
			)
			var darwinBuild = artifact.Artifact{
				Goos:   "darwin",
				Goarch: "amd64",
				Name:   "mybin",
				Path:   filepath.Join(dist, "darwinamd64", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"ID":     "default",
				},
			}
			var windowsBuild = artifact.Artifact{
				Goos:   "windows",
				Goarch: "amd64",
				Name:   "mybin.exe",
				Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary":    "mybin",
					"Extension": ".exe",
					"ID":        "default",
				},
			}
			ctx.Artifacts.Add(darwinBuild)
			ctx.Artifacts.Add(windowsBuild)
			ctx.Version = "0.0.1"
			ctx.Git.CurrentTag = "v0.0.1"
			ctx.Config.Archives[0].Format = format
			require.NoError(tt, Pipe{}.Run(ctx))
			var archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive))
			for _, arch := range archives.List() {
				require.Equal(t, "defaultarch", arch.Extra["ID"].(string), "all archives should have the archive ID set")
			}
			require.Len(tt, archives.List(), 2)
			darwin := archives.Filter(artifact.ByGoos("darwin")).List()[0]
			windows := archives.Filter(artifact.ByGoos("windows")).List()[0]
			require.Equal(tt, "foobar_0.0.1_darwin_amd64."+format, darwin.Name)
			require.Equal(tt, "foobar_0.0.1_windows_amd64.zip", windows.Name)

			require.Equal(t, []artifact.Artifact{darwinBuild}, darwin.Extra["Builds"].([]artifact.Artifact))
			require.Equal(t, []artifact.Artifact{windowsBuild}, windows.Extra["Builds"].([]artifact.Artifact))

			if format == "tar.gz" {
				// Check archive contents
				require.Equal(
					t,
					[]string{
						"README.md",
						"foo/bar",
						"foo/bar/foobar",
						"foo/bar/foobar/blah.txt",
						"mybin",
					},
					tarFiles(t, filepath.Join(dist, "foobar_0.0.1_darwin_amd64.tar.gz")),
				)
			}
			if format == "zip" {
				require.Equal(
					t,
					[]string{
						"README.md",
						"foo/bar/foobar/blah.txt",
						"mybin.exe",
					},
					zipFiles(t, filepath.Join(dist, "foobar_0.0.1_windows_amd64.zip")),
				)
			}
		})
	}
}

func zipFiles(t *testing.T, path string) []string {
	f, err := os.Open(path)
	require.NoError(t, err)
	info, err := f.Stat()
	require.NoError(t, err)
	r, err := zip.NewReader(f, info.Size())
	require.NoError(t, err)
	var paths = make([]string, len(r.File))
	for i, zf := range r.File {
		paths[i] = zf.Name
	}
	return paths
}

func tarFiles(t *testing.T, path string) []string {
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()
	var r = tar.NewReader(gr)
	var paths []string
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		paths = append(paths, next.Name)
	}
	return paths
}

func TestRunPipeBinary(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Format:       "binary",
					NameTemplate: defaultBinaryNameTemplate,
					Builds:       []string{"default"},
				},
			},
		},
	)
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"Ext":    ".exe",
			"ID":     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	var binaries = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
	darwin := binaries.Filter(artifact.ByGoos("darwin")).List()[0]
	windows := binaries.Filter(artifact.ByGoos("windows")).List()[0]
	require.Equal(t, "mybin_0.0.1_darwin_amd64", darwin.Name)
	require.Equal(t, "mybin_0.0.1_windows_amd64.exe", windows.Name)
	require.Len(t, binaries.List(), 2)
}

func TestRunPipeDistRemoved(t *testing.T) {
	var ctx = context.New(
		config.Project{
			Dist: "/path/nope",
			Archives: []config.Archive{
				{
					NameTemplate: "nope",
					Format:       "zip",
					Builds:       []string{"default"},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join("/path/to/nope", "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary":    "mybin",
			"Extension": ".exe",
			"ID":        "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to create directory /path/nope/nope.zip: open /path/nope/nope.zip: no such file or directory`)
}

func TestRunPipeInvalidGlob(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []string{
						"[x-]",
					},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to find files to archive: globbing failed for pattern [x-]: file does not exist`)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo{{ .fff }",
					Format:       "zip",
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeInvalidWrapInDirectoryTemplate(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:          []string{"default"},
					NameTemplate:    "foo",
					WrapInDirectory: "foo{{ .fff }",
					Format:          "zip",
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWrap(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:          []string{"default"},
					NameTemplate:    "foo",
					WrapInDirectory: "foo_{{ .Os }}",
					Format:          "tar.gz",
					Replacements: map[string]string{
						"darwin": "macOS",
					},
					Files: []string{
						"README.*",
					},
				},
			},
		},
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))

	// Check archive contents
	f, err := os.Open(filepath.Join(dist, "foo.tar.gz"))
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() { require.NoError(t, gr.Close()) }()
	r := tar.NewReader(gr)
	for _, n := range []string{"README.md", "mybin"} {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Equal(t, filepath.Join("foo_macOS", n), h.Name)
	}
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archives: []config.Archive{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.NotEmpty(t, ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Format)
	require.NotEmpty(t, ctx.Config.Archives[0].Files)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []string{
						"foo",
					},
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "zip", ctx.Config.Archives[0].Format)
	require.Equal(t, "foo", ctx.Config.Archives[0].Files[0])
}

func TestDefaultFormatBinary(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Format: "binary",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBinaryNameTemplate, ctx.Config.Archives[0].NameTemplate)
}

func TestFormatFor(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					Builds: []string{"default"},
					Format: "tar.gz",
					FormatOverrides: []config.FormatOverride{
						{
							Goos:   "windows",
							Format: "zip",
						},
					},
				},
			},
		},
	}
	require.Equal(t, "zip", packageFormat(ctx.Config.Archives[0], "windows"))
	require.Equal(t, "tar.gz", packageFormat(ctx.Config.Archives[0], "linux"))
}

func TestBinaryOverride(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(tt *testing.T) {
			var ctx = context.New(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							Builds:       []string{"default"},
							NameTemplate: defaultNameTemplate,
							Files: []string{
								"README.*",
							},
							FormatOverrides: []config.FormatOverride{
								{
									Goos:   "windows",
									Format: "binary",
								},
							},
						},
					},
				},
			)
			ctx.Git.CurrentTag = "v0.0.1"
			ctx.Artifacts.Add(artifact.Artifact{
				Goos:   "darwin",
				Goarch: "amd64",
				Name:   "mybin",
				Path:   filepath.Join(dist, "darwinamd64", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"ID":     "default",
				},
			})
			ctx.Artifacts.Add(artifact.Artifact{
				Goos:   "windows",
				Goarch: "amd64",
				Name:   "mybin.exe",
				Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"Binary": "mybin",
					"Ext":    ".exe",
					"ID":     "default",
				},
			})
			ctx.Version = "0.0.1"
			ctx.Config.Archives[0].Format = format

			require.NoError(tt, Pipe{}.Run(ctx))
			var archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive))
			darwin := archives.Filter(artifact.ByGoos("darwin")).List()[0]
			require.Equal(tt, "foobar_0.0.1_darwin_amd64."+format, darwin.Name)
			require.Equal(tt, format, darwin.ExtraOr("Format", ""))

			archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
			windows := archives.Filter(artifact.ByGoos("windows")).List()[0]
			require.Equal(tt, "foobar_0.0.1_windows_amd64.exe", windows.Name)
			require.Equal(tt, format, windows.ExtraOr("Format", ""))
		})
	}
}

func TestRunPipeSameArchiveFilename(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0755))
	_, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist:        dist,
			ProjectName: "foobar",
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "same-filename",
					Files: []string{
						"README.*",
						"./foo/**/*",
					},
					Format: "tar.gz",
				},
			},
		},
	)
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary": "mybin",
			"ID":     "default",
		},
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			"Binary":    "mybin",
			"Extension": ".exe",
			"ID":        "default",
		},
	})
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	err = Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "same-filename.tar.gz already exists. Check your archive name template")
}

func TestDuplicateFilesInsideArchive(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())

	ff, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer ff.Close()
	defer os.Remove(ff.Name())

	a := NewEnhancedArchive(archive.New(f), "")
	defer a.Close()
	require.NoError(t, a.Add("foo", ff.Name()))
	require.EqualError(t, a.Add("foo", ff.Name()), "file foo already exists in the archive")
}

func TestWrapInDirectory(t *testing.T) {
	t.Run("false", func(t *testing.T) {
		require.Equal(t, "", wrapFolder(config.Archive{
			WrapInDirectory: "false",
		}))
	})
	t.Run("true", func(t *testing.T) {
		require.Equal(t, "foo", wrapFolder(config.Archive{
			WrapInDirectory: "true",
			NameTemplate:    "foo",
		}))
	})
	t.Run("custom", func(t *testing.T) {
		require.Equal(t, "foobar", wrapFolder(config.Archive{
			WrapInDirectory: "foobar",
		}))
	})
}

func TestSeveralArchivesWithTheSameID(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Archives: []config.Archive{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 items with the ID 'a', please fix your config")
}
