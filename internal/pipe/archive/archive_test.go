package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/archive"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
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
		Formats []string
		Strip   bool
	}{
		{
			Formats: []string{"tar.gz", "zip"},
			Strip:   true,
		},
		{
			Formats: []string{"tar.gz", "zip"},
			Strip:   false,
		},
	} {
		formats := dets.Formats
		name := "archive." + strings.Join(formats, ",")
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
			f, err := os.Create(filepath.Join(folder, "foo", "bar", "foobar", "blah.txt"))
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
							Formats:              formats,
							NameTemplate:         defaultNameTemplate,
							StripBinaryDirectory: dets.Strip,
							Files: []config.File{
								{Source: "README.{{.Os}}.*"},
								{Source: "./foo/**/*"},
							},
							FormatOverrides: []config.FormatOverride{
								{
									Goos:    "windows",
									Formats: []string{"zip"},
								},
								{
									Goos:    "freebsd",
									Formats: []string{"none"},
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
				Extra: map[string]any{
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
				Extra: map[string]any{
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
				Extra: map[string]any{
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
				Extra: map[string]any{
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
				Extra: map[string]any{
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
				Extra: map[string]any{
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
				Extra: map[string]any{
					artifact.ExtraBinary: "mybin",
					artifact.ExtraID:     "default",
				},
			}
			freebsdAmd64Build := &artifact.Artifact{
				Goos:    "freebsd",
				Goarch:  "amd64",
				Goamd64: "v3",
				Name:    "bin/mybin",
				Path:    "will be ignored",
				Type:    artifact.Binary,
				Extra: map[string]any{
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
			ctx.Artifacts.Add(freebsdAmd64Build)
			ctx.Version = "0.0.1"
			ctx.Git.CurrentTag = "v0.0.1"
			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))

			require.Empty(t, ctx.Artifacts.Filter(
				artifact.And(
					artifact.ByGoos("freebsd"),
					artifact.Or(
						artifact.ByType(artifact.UploadableArchive),
						artifact.ByType(artifact.UploadableBinary),
					),
				),
			).List(), "shouldn't have archived freebsd in any way")

			archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
			for _, arch := range archives {
				expectBin := "bin/mybin"
				if arch.Goos == "windows" {
					expectBin += ".exe"
				}
				require.Equal(t, "myid", arch.ID(), "all archives must have the archive ID set")
				require.Equal(t, []string{expectBin}, artifact.MustExtra[[]string](*arch, artifact.ExtraBinaries))
				require.Empty(t, artifact.ExtraOr(*arch, artifact.ExtraBinary, ""))
			}
			require.Len(t, archives, 13)
			// TODO: should verify the artifact fields here too

			expectBin := "bin/mybin"
			if dets.Strip {
				expectBin = "mybin"
			}

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
				require.EqualValues(t, 0o755, header.Mode)
			}

			name := "foobar_0.0.1_windows_amd64.zip"
			require.Equal(
				t,
				[]string{
					"README.windows.md",
					"foo/bar/foobar/blah.txt",
					expectBin + ".exe",
				},
				testlib.LsArchive(t, filepath.Join(dist, name), "zip"),
			)
			info := zipInfo(t, filepath.Join(dist, name), expectBin+".exe")
			require.Equal(t, fs.FileMode(0o755), info.Mode())
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
				Formats:      []string{"tar.gz"},
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
		Extra: map[string]any{
			artifact.ExtraBinary: "bin/mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
}

func zipInfo(t *testing.T, path, name string) fs.FileInfo {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	info, err := f.Stat()
	require.NoError(t, err)
	r, err := zip.NewReader(f, info.Size())
	require.NoError(t, err)
	for _, next := range r.File {
		if next.Name == name {
			return next.FileInfo()
		}
	}
	t.Fatalf("could not find %q in %q", name, path)
	return nil
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
	t.Fatalf("could not find %q in %q", name, path)
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
					Formats:      []string{"binary"},
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
		Extra: map[string]any{
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
	require.True(t, artifact.MustExtra[bool](*darwinUniversal, artifact.ExtraReplaces))
	windows := binaries.Filter(artifact.ByGoos("windows")).List()[0]
	windows2 := binaries.Filter(artifact.ByGoos("windows")).List()[1]
	require.Equal(t, "mybin_0.0.1_darwin_amd64", darwinThin.Name)
	require.Equal(t, "mybin", artifact.MustExtra[string](*darwinThin, artifact.ExtraBinary))
	testlib.RequireNoExtraField(t, darwinThin, artifact.ExtraReplaces)
	require.Equal(t, "myunibin_0.0.1_darwin_all", darwinUniversal.Name)
	require.Equal(t, "myunibin", artifact.MustExtra[string](*darwinUniversal, artifact.ExtraBinary))
	require.Equal(t, "mybin_0.0.1_windows_amd64.exe", windows.Name)
	testlib.RequireNoExtraField(t, windows, artifact.ExtraReplaces)
	require.Equal(t, "mybin.exe", artifact.MustExtra[string](*windows, artifact.ExtraBinary))
	require.Equal(t, "myotherbin_0.0.1_windows_amd64.exe", windows2.Name)
	require.Equal(t, "myotherbin.exe", artifact.MustExtra[string](*windows2, artifact.ExtraBinary))
	testlib.RequireNoExtraField(t, windows2, artifact.ExtraReplaces)
}

func TestRunPipeDistRemoved(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: "/tmp/path/to/nope",
			Archives: []config.Archive{
				{
					NameTemplate: "nope",
					Formats:      []string{"zip"},
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
		Extra: map[string]any{
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
					Formats:      []string{"zip"},
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), `failed to find files to archive: globbing failed for pattern [x-]: compile glob pattern: unexpected end of input`)
}

func TestRunPipeNameTemplateWithSpace(t *testing.T) {
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
					NameTemplate: " foo_{{.Os}}_{{.Arch}} ",
					Formats:      []string{"zip"},
				},
				{
					Builds:       []string{"default"},
					NameTemplate: " foo_{{.Os}}_{{.Arch}} ",
					Formats:      []string{"binary"},
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary)).List()
	require.Len(t, list, 1)
	require.Equal(t, "foo_darwin_amd64", list[0].Name)
	list = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
	require.Len(t, list, 1)
	require.Equal(t, "foo_darwin_amd64.zip", list[0].Name)
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
					Formats:      []string{"zip"},
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
		Extra: map[string]any{
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
					Formats:      []string{"zip"},
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
		Extra: map[string]any{
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
					Formats:         []string{"zip"},
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
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
					Formats:         []string{"tar.gz"},
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraID:     "default",
		},
	})
	require.NoError(t, Pipe{}.Run(ctx))

	archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive)).List()
	require.Len(t, archives, 1)
	require.Equal(t, "foo_darwin", artifact.MustExtra[string](*archives[0], artifact.ExtraWrappedIn))

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
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Formats[0])
	require.NotEmpty(t, ctx.Config.Archives[0].Files)
	require.Equal(t, fs.FileMode(0o755), ctx.Config.Archives[0].BuildsInfo.Mode)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Builds:       []string{"default"},
				NameTemplate: "foo",
				Formats:      []string{"zip"},
				Files: []config.File{
					{Source: "foo"},
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Archives[0].NameTemplate)
	require.Equal(t, "zip", ctx.Config.Archives[0].Formats[0])
	require.Equal(t, config.File{Source: "foo"}, ctx.Config.Archives[0].Files[0])
}

func TestDefaultMixFormats(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Formats: []string{"tar.gz", "binary"},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultBinaryNameTemplate, ctx.Config.Archives[0].NameTemplate)
}

func TestDefaultNoFiles(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Formats: []string{"tar.gz"},
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
				Formats: []string{"binary"},
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
				Builds:  []string{"default"},
				Formats: []string{"tar.gz", "tar.xz"},
				FormatOverrides: []config.FormatOverride{
					{
						Goos:    "windows",
						Formats: []string{"zip", "7z"},
					},
					{
						Goos:    "darwin",
						Formats: []string{"none"},
					},
				},
			},
		},
	})
	require.Equal(t, []string{"zip", "7z"}, packageFormats(ctx.Config.Archives[0], "windows"))
	require.Equal(t, []string{"tar.gz", "tar.xz"}, packageFormats(ctx.Config.Archives[0], "linux"))
	require.Equal(t, []string{"none"}, packageFormats(ctx.Config.Archives[0], "darwin"))
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
					Formats: []string{"tar.gz", "zip"},
					FormatOverrides: []config.FormatOverride{
						{
							Goos:    "windows",
							Formats: []string{"binary"},
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
		Extra: map[string]any{
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Version = "0.0.1"

	require.NoError(t, Pipe{}.Run(ctx))
	archives := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableArchive))

	darwins := archives.Filter(artifact.ByGoos("darwin")).List()
	require.Len(t, darwins, 2)
	for _, darwin := range darwins {
		format := darwin.Format()
		require.Contains(t, []string{"tar.gz", "zip"}, format)
		require.Equal(t, "foobar_0.0.1_darwin_amd64."+format, darwin.Name)
		require.Empty(t, artifact.ExtraOr(*darwin, artifact.ExtraWrappedIn, ""))
	}

	archives = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary))
	windows := archives.Filter(artifact.ByGoos("windows")).List()[0]
	require.Equal(t, "foobar_0.0.1_windows_amd64.exe", windows.Name)
	require.Empty(t, artifact.ExtraOr(*windows, artifact.ExtraWrappedIn, ""))
	require.Equal(t, "mybin.exe", artifact.MustExtra[string](*windows, artifact.ExtraBinary))
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
					Formats: []string{"tar.gz"},
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
		Extra: map[string]any{
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
		Extra: map[string]any{
			artifact.ExtraBinary: "mybin",
			artifact.ExtraExt:    ".exe",
			artifact.ExtraID:     "default",
		},
	})
	ctx.Version = "0.0.1"
	ctx.Git.CurrentTag = "v0.0.1"
	err = Pipe{}.Run(ctx)
	require.ErrorContains(t, err, "same-filename.tar.gz already exists. Check your archive name template")
}

func TestDuplicateFilesInsideArchive(t *testing.T) {
	folder := t.TempDir()

	f, err := os.CreateTemp(folder, "")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

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
		require.Empty(t, wrapFolder(config.Archive{
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
		t.Cleanup(func() { require.NoError(t, bin.Close()) })
		dist := t.TempDir()
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Builds:       []string{"default"},
					Formats:      []string{"tar.gz"},
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
			Extra: map[string]any{
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
				Formats:      []string{"7z"},
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
				Formats:      []string{"zip"},
				Files: []config.File{
					{Source: "./testdata/a/a.txt"},
				},
			},
			{
				ID:           "foobar",
				NameTemplate: "foobar",
				Meta:         true,
				Formats:      []string{"zip"},
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
				Formats:      []string{"tar.gz"},
				FormatOverrides: []config.FormatOverride{{
					Goos:    "windows",
					Formats: []string{"zip"},
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
		Extra: map[string]any{
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

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.New(testctx.Skip(skips.Archive))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.New()))
	})
}

func TestDefaultDeprecatd(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Archives: []config.Archive{
			{
				Format: "tar.gz",
				FormatOverrides: []config.FormatOverride{
					{
						Format: "zip",
					},
				},
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, ctx.Deprecated)
	require.Equal(t, "tar.gz", ctx.Config.Archives[0].Formats[0])
	require.Equal(t, "zip", ctx.Config.Archives[0].FormatOverrides[0].Formats[0])
}
