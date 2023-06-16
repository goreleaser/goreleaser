package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/archive"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func createFakeBinary(t *testing.T, dist, arch, bin string) {
	t.Helper()
	path := filepath.Join(dist, arch, bin)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestRunPipe(t *testing.T) {
	folder := testlib.Mktmp(t)
	for _, dets := range []struct {
		Format string
		Strip  bool
	}{
		{
			Format: "tar.gz",
			Strip:  true,
		},
		{
			Format: "tar.gz",
			Strip:  false,
		},

		{
			Format: "zip",
			Strip:  true,
		},
		{
			Format: "zip",
			Strip:  false,
		},
	} {
		format := dets.Format
		name := "archive." + format
		if dets.Strip {
			name = "strip_" + name
		}
		t.Run(name, func(t *testing.T) {
			dist := filepath.Join(folder, name+"_dist")
			require.NoError(t, os.Mkdir(dist, 0o755))
			for _, arch := range []string{"darwinamd64v1", "darwinall", "linux386", "linuxarm7", "linuxmipssoftfloat", "linuxamd64v3"} {
				createFakeBinary(t, dist, arch, "bin/mybin")
			}
			createFakeBinary(t, dist, "windowsamd64", "bin/mybin.exe")
			for _, tt := range []string{"darwin", "linux", "windows"} {
				f, err := os.Create(filepath.Join(folder, fmt.Sprintf("README.%s.md", tt)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			require.NoError(t, os.MkdirAll(filepath.Join(folder, "foo", "bar", "foobar"), 0o755))
			f, err := os.Create(filepath.Join(filepath.Join(folder, "foo", "bar", "foobar", "blah.txt")))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							ID:     "myid",
							Builds: []string{"default"},
							BuildsInfo: config.FileInfo{
								Owner: "root",
								Group: "root",
							},
							NameTemplate:            defaultNameTemplate,
							StripParentBinaryFolder: dets.Strip,
							Files: []config.File{
								{Source: "README.{{.Os}}.*"},
								{Source: "./foo/**/*"},
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
			darwinUniversalBinary := &artifact.Artifact{
				Goos:   "darwin",
				Goarch: "all",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "darwinall", "bin", "mybin"),
				Type:   artifact.UniversalBinary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary:   "bin/mybin",
					artifact.ExtraID:       "default",
					artifact.ExtraReplaces: true,
				},
			}
			darwinBuild := &artifact.Artifact{
				Goos:    "darwin",
				Goarch:  "amd64",
				Goamd64: "v1",
				Name:    "bin/mybin",
				Path:    filepath.Join(dist, "darwinamd64v1", "bin", "mybin"),
				Type:    artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "bin/mybin",
					artifact.ExtraID:     "default",
				},
			}
			linux386Build := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "386",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linux386", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "bin/mybin",
					artifact.ExtraID:     "default",
				},
			}
			linuxArmBuild := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "arm",
				Goarm:  "7",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linuxarm7", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "bin/mybin",
					artifact.ExtraID:     "default",
				},
			}
			linuxMipsBuild := &artifact.Artifact{
				Goos:   "linux",
				Goarch: "mips",
				Gomips: "softfloat",
				Name:   "bin/mybin",
				Path:   filepath.Join(dist, "linuxmipssoftfloat", "bin", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraID:     "default",
				},
			}
			windowsBuild := &artifact.Artifact{
				Goos:    "windows",
				Goarch:  "amd64",
				Goamd64: "v1",
				Name:    "bin/mybin.exe",
				Path:    filepath.Join(dist, "windowsamd64", "bin", "mybin.exe"),
				Type:    artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraExt:    ".exe",
					artifact.ExtraID:     "default",
				},
			}
			linuxAmd64Build := &artifact.Artifact{
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v3",
				Name:    "bin/mybin",
				Path:    filepath.Join(dist, "linuxamd64v3", "bin", "mybin"),
				Type:    artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraID:     "default",
				},
			}
			ctx.Artifacts.Add(darwinBuild)
			ctx.Artifacts.Add(darwinUniversalBinary)
			ctx.Artifacts.Add(linux386Build)
			ctx.Artifacts.Add(linuxArmBuild)
			ctx.Artifacts.Add(linuxMipsBuild)
			ctx.Artifacts.Add(windowsBuild)
			ctx.Artifacts.Add(linuxAmd64Build)
			ctx.Version = "0.0.1"
			ctx.Git.CurrentTag = "v0.0.1"
			ctx.Config.Archives[0].Format = format
			require.NoError(t, Pipe{}.Run(ctx))
			archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()

			for _, arch := range archives {
				expectBin := "bin/mybin"
				if arch.Goos == "windows" {
					expectBin += ".exe"
				}
				require.Equal(t, "myid", arch.ID(), "all archives must have the archive ID set")
				require.Equal(t, []string{expectBin}, artifact.ExtraOr(*arch, artifact.ExtraBinaries, []string{}))
				require.Equal(t, "", artifact.ExtraOr(*arch, artifact.ExtraBinary, ""))
			}
			require.Len(t, archives, 7)
			// TODO: should verify the artifact fields here too

			expectBin := "bin/mybin"
			if dets.Strip {
				expectBin = "mybin"
			}

			if format == "tar.gz" {
				// Check archive contents
				for name, os := range map[string]string{
					"foobar_0.0.1_darwin_amd64.tar.gz":         "darwin",
					"foobar_0.0.1_darwin_all.tar.gz":           "darwin",
					"foobar_0.0.1_linux_386.tar.gz":            "linux",
					"foobar_0.0.1_linux_armv7.tar.gz":          "linux",
					"foobar_0.0.1_linux_mips_softfloat.tar.gz": "linux",
					"foobar_0.0.1_linux_amd64v3.tar.gz":        "linux",
				} {
					require.Equal(
						t,
						[]string{
							fmt.Sprintf("README.%s.md", os),
							"foo/bar/foobar/blah.txt",
							expectBin,
						},
						testlib.LsArchive(t, filepath.Join(dist, name), "tar.gz"),
					)

					header := tarInfo(t, filepath.Join(dist, name), expectBin)
					require.Equal(t, "root", header.Uname)
					require.Equal(t, "root", header.Gname)

				}
			}
			if format == "zip" {
				require.Equal(
					t,
					[]string{
						"README.windows.md",
						"foo/bar/foobar/blah.txt",
						expectBin + ".exe",
					},
					testlib.LsArchive(t, filepath.Join(dist, "foobar_0.0.1_windows_amd64.zip"), "zip"),
				)
			}
		})
	}
}

func TestRunPipeDifferentBinaryCount(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	for _, arch := range []string{"darwinamd64", "linuxamd64"} {
		createFakeBinary(t, dist, arch, "bin/mybin")
	}
	createFakeBinary(t, dist, "darwinamd64", "bin/foobar")
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        dist,
		ProjectName: "foobar",
		Archives: []config.Archive{
			{
				ID:           "myid",
				Format:       "tar.gz",
				Builds:       []string{"default", "foobar"},
				NameTemplate: defaultNameTemplate,
			},
		},
	})
	darwinBuild := &artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "darwinamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "default",
		},
	}
	darwinBuild2 := &artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "bin/foobar",
		Path:   filepath.Join(dist, "darwinamd64", "bin", "foobar"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "bin/foobar",
			artifact.ExtraID:     "foobar",
		},
	}
	linuxArmBuild := &artifact.Artifact{
		Goos:   "linux",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linuxamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "default",
		},
	}

	ctx.Artifacts.Add(darwinBuild)
	ctx.Artifacts.Add(darwinBuild2)
	ctx.Artifacts.Add(linuxArmBuild)
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"

	t.Run("check enabled", func(t *testing.T) {
		ctx.Config.Archives[0].AllowDifferentBinaryCount = false
		require.EqualError(t, Pipe{}.Run(ctx), "invalid archive: 0: "+ErrArchiveDifferentBinaryCount.Error())
	})

	t.Run("check disabled", func(t *testing.T) {
		ctx.Config.Archives[0].AllowDifferentBinaryCount = true
		require.NoError(t, Pipe{}.Run(ctx))
	})
}

func TestRunPipeNoBinaries(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        dist,
		ProjectName: "foobar",
		Archives: []config.Archive{{
			Builds: []string{"not-default"},
		}},
	}, testctx.WithVersion("0.0.1"), testctx.WithCurrentTag("v1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "linux",
		Goarch: "amd64",
		Name:   "bin/mybin",
		Path:   filepath.Join(dist, "linuxamd64", "bin", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
}

func tarInfo(t *testing.T, path, name string) *tar.Header {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()
	r := tar.NewReader(gr)
	for {
		next, err := r.Next()
		if err == io.EOF {
			break
		}
		if next.Name == name {
			return next
		}
	}
	return nil
}

func TestRunPipeBinary(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "myotherbin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Format:       "binary",
					NameTemplate: defaultBinaryNameTemplate,
					Builds:       []string{"default", "default2"},
				},
			},
		},
		testctx.WithVersion("0.0.1"),
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "all",
		Name:   "myunibin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.UniversalBinary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary:   "myunibin",
			artifact.ExtraID:       "default",
			artifact.ExtraReplaces: true,
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "myotherbin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "myotherbin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "myotherbin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default2",
		},
	})

	require.NoError(t, Pipe{}.Run(ctx))
	binaries := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
	require.Len(t, binaries.List(), 4)
	darwinThin := binaries.Filter(artifact.And(
		artifact.ByGoos("darwin"),
		artifact.ByGoarch("amd64"),
	)).List()[0]
	darwinUniversal := binaries.Filter(artifact.And(
		artifact.ByGoos("darwin"),
		artifact.ByGoarch("all"),
	)).List()[0]
	require.True(t, artifact.ExtraOr(*darwinUniversal, artifact.ExtraReplaces, false))
	windows := binaries.Filter(artifact.ByGoos("windows")).List()[0]
	windows2 := binaries.Filter(artifact.ByGoos("windows")).List()[1]
	require.Equal(t, "mybin_0.0.1_darwin_amd64", darwinThin.Name)
	require.Equal(t, "mybin", artifact.ExtraOr(*darwinThin, artifact.ExtraBinary, ""))
	require.Equal(t, "myunibin_0.0.1_darwin_all", darwinUniversal.Name)
	require.Equal(t, "myunibin", artifact.ExtraOr(*darwinUniversal, artifact.ExtraBinary, ""))
	require.Equal(t, "mybin_0.0.1_windows_amd64.exe", windows.Name)
	require.Equal(t, "mybin.exe", artifact.ExtraOr(*windows, artifact.ExtraBinary, ""))
	require.Equal(t, "myotherbin_0.0.1_windows_amd64.exe", windows2.Name)
	require.Equal(t, "myotherbin.exe", artifact.ExtraOr(*windows2, artifact.ExtraBinary, ""))
}

func TestRunPipeDistRemoved(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: "/tmp/path/to/nope",
			Archives: []config.Archive{
				{
					NameTemplate: "nope",
					Format:       "zip",
					Builds:       []string{"default"},
				},
			},
		},
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join("/tmp/path/to/nope", "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	})
	// not checking on error msg because it may change depending on OS/version
	require.Error(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidGlob(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []config.File{
						{Source: "[x-]"},
					},
				},
			},
		},
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Git.CurrentTag = "v0.0.1"
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to find files to archive: globbing failed for pattern [x-]: compile glob pattern: unexpected end of input`)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
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
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidFilesNameTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "foo",
					Format:       "zip",
					Files: []config.File{
						{Source: "{{.asdsd}"},
					},
				},
			},
		},
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestRunPipeInvalidWrapInDirectoryTemplate(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
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
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWrap(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:          []string{"default"},
					NameTemplate:    "foo",
					WrapInDirectory: "foo_{{ .Os }}",
					Format:          "tar.gz",
					Files: []config.File{
						{Source: "README.*"},
					},
				},
			},
		},
		testctx.WithCurrentTag("v0.0.1"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join("dist", "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))

	archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
	require.Len(t, archives, 1)
	require.Equal(t, "foo_darwin", artifact.ExtraOr(*archives[0], artifact.ExtraWrappedIn, ""))

	require.ElementsMatch(
		t,
		[]string{"foo_darwin/README.md", "foo_darwin/mybin"},
		testlib.LsArchive(t, filepath.Join(dist, "foo.tar.gz"), "tar.gz"),
	)
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NotEmpty(t, ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Format)
	require.NotEmpty(t, ctx.Config.Archives[0].Files)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Builds:       []string{"default"},
				NameTemplate: "foo",
				Format:       "zip",
				Files: []config.File{
					{Source: "foo"},
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "zip", ctx.Config.Archives[0].Format)
	require.Equal(t, config.File{Source: "foo"}, ctx.Config.Archives[0].Files[0])
}

func TestDefaultNoFiles(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Format: "tar.gz",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Archives[0].NameTemplate)
}

func TestDefaultFormatBinary(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Format: "binary",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBinaryNameTemplate, ctx.Config.Archives[0].NameTemplate)
}

func TestFormatFor(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
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
	})
	require.Equal(t, "zip", packageFormat(ctx.Config.Archives[0], "windows"))
	require.Equal(t, "tar.gz", packageFormat(ctx.Config.Archives[0], "linux"))
}

func TestBinaryOverride(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(folder, "README.md"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run("Archive format "+format, func(t *testing.T) {
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        dist,
					ProjectName: "foobar",
					Archives: []config.Archive{
						{
							Builds:       []string{"default"},
							NameTemplate: defaultNameTemplate,
							Files: []config.File{
								{Source: "README.*"},
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
				testctx.WithCurrentTag("v0.0.1"),
			)
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "darwin",
				Goarch: "amd64",
				Name:   "mybin",
				Path:   filepath.Join(dist, "darwinamd64", "mybin"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraID:     "default",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Goos:   "windows",
				Goarch: "amd64",
				Name:   "mybin.exe",
				Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraExt:    ".exe",
					artifact.ExtraID:     "default",
				},
			})
			ctx.Version = "0.0.1"
			ctx.Config.Archives[0].Format = format

			require.NoError(t, Pipe{}.Run(ctx))
			archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive))
			darwin := archives.Filter(artifact.ByGoos("darwin")).List()[0]
			require.Equal(t, "foobar_0.0.1_darwin_amd64."+format, darwin.Name)
			require.Equal(t, format, darwin.Format())
			require.Empty(t, artifact.ExtraOr(*darwin, artifact.ExtraWrappedIn, ""))

			archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
			windows := archives.Filter(artifact.ByGoos("windows")).List()[0]
			require.Equal(t, "foobar_0.0.1_windows_amd64.exe", windows.Name)
			require.Empty(t, artifact.ExtraOr(*windows, artifact.ExtraWrappedIn, ""))
			require.Equal(t, "mybin.exe", artifact.ExtraOr(*windows, artifact.ExtraBinary, ""))
		})
	}
}

func TestRunPipeSameArchiveFilename(t *testing.T) {
	folder := testlib.Mktmp(t)
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "darwinamd64"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "windowsamd64"), 0o755))
	f, err := os.Create(filepath.Join(dist, "darwinamd64", "mybin"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Create(filepath.Join(dist, "windowsamd64", "mybin.exe"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dist,
			ProjectName: "foobar",
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					NameTemplate: "same-filename",
					Files: []config.File{
						{Source: "README.*"},
						{Source: "./foo/**/*"},
					},
					Format: "tar.gz",
				},
			},
		},
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "darwin",
		Goarch: "amd64",
		Name:   "mybin",
		Path:   filepath.Join(dist, "darwinamd64", "mybin"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Goos:   "windows",
		Goarch: "amd64",
		Name:   "mybin.exe",
		Path:   filepath.Join(dist, "windowsamd64", "mybin.exe"),
		Type:   artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	err = Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "same-filename.tar.gz already exists. Check your archive name template")
}

func TestDuplicateFilesInsideArchive(t *testing.T) {
	folder := t.TempDir()

	f, err := os.CreateTemp(folder, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

	ff, err := os.CreateTemp(folder, "")
	require.NoError(t, err)
	require.NoError(t, ff.Close())
	a, err := archive.New(f, "tar.gz")
	require.NoError(t, err)
	a = NewEnhancedArchive(a, "")
	t.Cleanup(func() {
		require.NoError(t, a.Close())
	})

	require.NoError(t, a.Add(config.File{
		Source:      ff.Name(),
		Destination: "foo",
	}))
	require.ErrorIs(t, a.Add(config.File{
		Source:      ff.Name(),
		Destination: "foo",
	}), fs.ErrExist)
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
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	})
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 archives with the ID 'a', please fix your config")
}

func TestArchive_globbing(t *testing.T) {
	assertGlob := func(t *testing.T, files []config.File, expected []string) {
		t.Helper()
		bin, err := os.CreateTemp(t.TempDir(), "binary")
		require.NoError(t, err)
		dist := t.TempDir()
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					Format:       "tar.gz",
					NameTemplate: "foo",
					Files:        files,
				},
			},
		})

		ctx.Artifacts.Add(&artifact.Artifact{
			Goos:   "darwin",
			Goarch: "amd64",
			Name:   "foobin",
			Path:   bin.Name(),
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				artifact.ExtraID: "default",
			},
		})

		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(t, append(expected, "foobin"), testlib.LsArchive(t, filepath.Join(dist, "foo.tar.gz"), "tar.gz"))
	}

	t.Run("exact src file", func(t *testing.T) {
		assertGlob(t, []config.File{{Source: "testdata/a/a.txt"}}, []string{"testdata/a/a.txt"})
	})

	t.Run("exact src file with dst", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/a/a.txt",
				Destination: "foo/",
			},
		}, []string{"foo"})
	})

	t.Run("glob src", func(t *testing.T) {
		assertGlob(t, []config.File{
			{Source: "testdata/**/*.txt"},
		}, []string{
			"testdata/a/a.txt",
			"testdata/a/b/a.txt",
			"testdata/a/b/c/d.txt",
		})
	})

	t.Run("glob src with dst", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/**/*.txt",
				Destination: "var/yada",
			},
		}, []string{
			"var/yada/a.txt",
			"var/yada/b/a.txt",
			"var/yada/b/c/d.txt",
		})
	})

	t.Run("glob src with dst stripping parent", func(t *testing.T) {
		assertGlob(t, []config.File{
			{
				Source:      "testdata/**/*.txt",
				Destination: "var/yada",
				StripParent: true,
			},
		}, []string{
			"var/yada/a.txt",
			"var/yada/d.txt",
		})
	})
}

func TestInvalidFormat(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist: t.TempDir(),
		Archives: []config.Archive{
			{
				ID:           "foo",
				NameTemplate: "foo",
				Meta:         true,
				Format:       "7z",
			},
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), "invalid archive format: 7z")
}

func TestIssue3803(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist: t.TempDir(),
		Archives: []config.Archive{
			{
				ID:           "foo",
				NameTemplate: "foo",
				Meta:         true,
				Format:       "zip",
				Files: []config.File{
					{Source: "./testdata/a/a.txt"},
				},
			},
			{
				ID:           "foobar",
				NameTemplate: "foobar",
				Meta:         true,
				Format:       "zip",
				Files: []config.File{
					{Source: "./testdata/a/b/a.txt"},
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	archives := ctx.Artifacts.List()
	require.Len(t, archives, 2)
}

func TestExtraFormatWhenOverride(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist: t.TempDir(),
		Archives: []config.Archive{
			{
				ID:           "foo",
				NameTemplate: "foo",
				Format:       "tar.gz",
				FormatOverrides: []config.FormatOverride{{
					Goos:   "windows",
					Format: "zip",
				}},
				Files: []config.File{
					{Source: "./testdata/a/a.txt"},
				},
			},
		},
	})
	windowsBuild := &artifact.Artifact{
		Goos:    "windows",
		Goarch:  "amd64",
		Goamd64: "v1",
		Name:    "bin/mybin.exe",
		Path:    filepath.Join(ctx.Config.Dist, "windowsamd64", "bin", "mybin.exe"),
		Type:    artifact.Binary,
		Extra: map[string]interface{}{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(windowsBuild.Path), 0o755))
	f, err := os.Create(windowsBuild.Path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(windowsBuild)
	require.NoError(t, Pipe{}.Run(ctx))
	archives := ctx.Artifacts.Filter(artifact.ByFormats("zip")).List()
	require.Len(t, archives, 1)
}
