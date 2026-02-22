//go:build integration

package nfpm

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRunPipe(t *testing.T) {
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.ToSlash(filepath.Join(dist, "mybin", "mybin"))
	foohPath := filepath.ToSlash(filepath.Join(dist, "foo.h"))
	foosoPath := filepath.ToSlash(filepath.Join(dist, "foo.so"))
	fooaPath := filepath.ToSlash(filepath.Join(dist, "foo.a"))
	for _, name := range []string{binPath, foosoPath, foohPath, fooaPath} {
		f, err := os.Create(name)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
	libPrefix := `/usr/lib
      {{- if eq .Arch "amd64" }}{{if eq .Format "rpm"}}_rpm{{end}}64{{- end -}}
	`
	now := time.Now().Truncate(time.Second).UTC()
	fileInfo := config.FileInfo{
		MTime: "{{.CommitDate}}",
	}
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Env: []string{
			"PRO=pro",
			"DESC=templates",
			"EXT=.sh",
		},
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Bindir:      "/usr/bin",
				Builds:      []string{"default", "lib1", "lib2", "lib3"},
				Formats:     []string{"deb", "rpm", "apk", "termux.deb", "archlinux", "ipk"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description with {{ .Env.DESC }}",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.com/{{ .Env.PRO }}",
				Changelog:   "./testdata/changelog.yaml",
				MTime:       "{{.CommitDate}}",
				GoAmd64:     []string{"v1", "v2", "v3", "v4"},
				Libdirs: config.Libdirs{
					Header:   libPrefix + "/headers",
					CArchive: libPrefix + "/c-archives",
					CShared:  libPrefix + "/c-shareds",
				},
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
					Scripts: config.NFPMScripts{
						PreInstall:  "./testdata/pre_install{{.Env.EXT}}",
						PostInstall: "./testdata/post_install{{.Env.EXT}}",
						PreRemove:   "./testdata/pre_remove{{.Env.EXT}}",
						PostRemove:  "./testdata/post_remove{{.Env.EXT}}",
					},
					Contents: []config.NFPMContent{
						{
							Destination: "/var/log/foobar",
							Type:        "dir",
							FileInfo:    fileInfo,
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/usr/share/testfile.txt",
							FileInfo:    fileInfo,
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope.conf",
							Type:        "config",
							FileInfo:    fileInfo,
						},
						{
							Destination: "/etc/mydir",
							Type:        "dir",
							FileInfo:    fileInfo,
						},
						{
							Source:      "./testdata/testfile.txt",
							Destination: "/etc/nope-rpm.conf",
							FileInfo:    fileInfo,
							Type:        "config",
							Packager:    "rpm",
						},
						{
							Source:      "/etc/nope.conf",
							Destination: "/etc/nope2.conf",
							FileInfo:    fileInfo,
							Type:        "symlink",
						},
						{
							Source:      "./testdata/testfile-{{ .Arch }}{{.Amd64}}{{.Arm}}{{.Mips}}.txt",
							Destination: "/etc/nope3_{{ .ProjectName }}.conf",
							FileInfo:    fileInfo,
						},
						{
							Source:      "./testdata/folder",
							Destination: "/etc/folder",
							FileInfo:    fileInfo,
						},
					},
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"), func(ctx *context.Context) {
		ctx.Git.CommitDate = now
	})

	for _, goos := range []string{"linux", "darwin", "ios", "android", "aix"} {
		for _, goarch := range []string{"amd64", "386", "arm64", "arm", "mips", "ppc64"} {
			if goos == "ios" && goarch != "arm64" {
				continue
			}
			if goarch == "ppc64" && goos != "aix" {
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
						Extra: map[string]any{
							artifact.ExtraID: "default",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.h",
						Path:   foohPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.Header,
						Extra: map[string]any{
							artifact.ExtraID: "lib1",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.so",
						Path:   foosoPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.CShared,
						Extra: map[string]any{
							artifact.ExtraID: "lib2",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.a",
						Path:   fooaPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.CArchive,
						Extra: map[string]any{
							artifact.ExtraID: "lib3",
						},
					})
				}
			case "amd64":
				// v5 is invalid, filtered out in tests
				for _, goamd64 := range []string{"v1", "v2", "v3", "v4", "v5"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "subdir/mybin",
						Path:    binPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.Binary,
						Extra: map[string]any{
							artifact.ExtraID: "default",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "foo.h",
						Path:    foohPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.Header,
						Extra: map[string]any{
							artifact.ExtraID: "lib1",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "foo.so",
						Path:    foosoPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.CShared,
						Extra: map[string]any{
							artifact.ExtraID: "lib2",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:    "foo.a",
						Path:    fooaPath,
						Goarch:  goarch,
						Goos:    goos,
						Goamd64: goamd64,
						Type:    artifact.CArchive,
						Extra: map[string]any{
							artifact.ExtraID: "lib3",
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
						Extra: map[string]any{
							artifact.ExtraID: "default",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.h",
						Path:   foohPath,
						Goarch: goarch,
						Goos:   goos,
						Gomips: gomips,
						Type:   artifact.Header,
						Extra: map[string]any{
							artifact.ExtraID: "lib1",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.so",
						Path:   foosoPath,
						Goarch: goarch,
						Goos:   goos,
						Gomips: gomips,
						Type:   artifact.CShared,
						Extra: map[string]any{
							artifact.ExtraID: "lib2",
						},
					})
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "foo.a",
						Path:   fooaPath,
						Goarch: goarch,
						Goos:   goos,
						Gomips: gomips,
						Type:   artifact.CArchive,
						Extra: map[string]any{
							artifact.ExtraID: "lib3",
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
					Extra: map[string]any{
						artifact.ExtraID: "default",
					},
				})
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "foo.h",
					Path:   foohPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Header,
					Extra: map[string]any{
						artifact.ExtraID: "lib1",
					},
				})
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "foo.so",
					Path:   foosoPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.CShared,
					Extra: map[string]any{
						artifact.ExtraID: "lib2",
					},
				})
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "foo.a",
					Path:   fooaPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.CArchive,
					Extra: map[string]any{
						artifact.ExtraID: "lib3",
					},
				})
			}
		}
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 57)

	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Equal(t, "."+pkg.Format(), pkg.Ext())
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
		if format != termuxFormat {
			packager, err := nfpm.Get(format)
			require.NoError(t, err)

			if packager, ok := packager.(nfpm.PackagerWithExtension); ok {
				ext = packager.ConventionalExtension()
			}
		}

		switch pkg.Goos {
		case "linux":
			require.Equal(t, "foo_1.0.0_linux_"+arch+"-10-20"+ext, pkg.Name)
		case "android":
			require.Equal(t, "foo_1.0.0_android_"+arch+"-10-20"+ext, pkg.Name)
		case "aix":
			require.Equal(t, "foo_1.0.0_aix_ppc64-10-20"+ext, pkg.Name)
		default:
			require.Equal(t, "foo_1.0.0_ios_arm64-10-20"+ext, pkg.Name)
		}
		require.Equal(t, "someid", pkg.ID())

		stat, err := os.Stat(pkg.Path)
		require.NoError(t, err)
		require.Equal(t, now.UTC(), stat.ModTime().UTC())

		contents := artifact.MustExtra[files.Contents](*pkg, extraFiles)
		for _, src := range contents {
			require.NotNil(t, src.FileInfo, src.Destination)
			require.Equal(t, now.UTC(), src.FileInfo.MTime.UTC(), src.Destination)
		}

		require.ElementsMatch(t, []string{
			"./testdata/testfile.txt",
			"./testdata/testfile.txt",
			"./testdata/testfile.txt",
			"/etc/nope.conf",
			"./testdata/folder",
			"./testdata/testfile-" + pkg.Goarch + pkg.Goamd64 + pkg.Goarm + pkg.Gomips + ".txt",
			binPath,
			foohPath,
			fooaPath,
			foosoPath,
		}, sources(contents))

		bin := "/usr/bin/subdir/"
		header := "/usr/lib/headers"
		carchive := "/usr/lib/c-archives"
		cshared := "/usr/lib/c-shareds"
		if pkg.Goarch == "amd64" {
			if pkg.Format() == "rpm" {
				header = "/usr/lib_rpm64/headers"
				carchive = "/usr/lib_rpm64/c-archives"
				cshared = "/usr/lib_rpm64/c-shareds"
			} else {
				header = "/usr/lib64/headers"
				carchive = "/usr/lib64/c-archives"
				cshared = "/usr/lib64/c-shareds"
			}
		}
		if format == termuxFormat {
			bin = filepath.Join("/data/data/com.termux/files", bin)
			header = filepath.Join("/data/data/com.termux/files", header)
			cshared = filepath.Join("/data/data/com.termux/files", cshared)
			carchive = filepath.Join("/data/data/com.termux/files", carchive)
		}
		bin = filepath.ToSlash(filepath.Join(bin, "mybin"))
		header = filepath.ToSlash(filepath.Join(header, "foo.h"))
		cshared = filepath.ToSlash(filepath.Join(cshared, "foo.so"))
		carchive = filepath.ToSlash(filepath.Join(carchive, "foo.a"))
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
			header,
			carchive,
			cshared,
		}, destinations(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
	}
	require.Len(t, ctx.Config.NFPMs[0].Contents, 8, "should not modify the config file list")
}

func TestIntegrationRunPipeConventionalNameTemplate(t *testing.T) {
	t.Run("regular", func(t *testing.T) { doTestRunPipeConventionalNameTemplate(t, false) })
	t.Run("snapshot", func(t *testing.T) { doTestRunPipeConventionalNameTemplate(t, true) })
}

func doTestRunPipeConventionalNameTemplate(t *testing.T, snapshot bool) {
	t.Helper()
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
	binPath := filepath.ToSlash(filepath.Join(dist, "mybin", "mybin"))
	require.NoError(t, os.WriteFile(binPath, []byte("nope"), 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		NFPMs: []config.NFPM{
			{
				ID:          "someid",
				Builds:      []string{"default"},
				Formats:     []string{"deb", "rpm", "apk", "archlinux", "ipk"},
				Section:     "somesection",
				Priority:    "standard",
				Description: "Some description ",
				License:     "MIT",
				Maintainer:  "me@me",
				Vendor:      "asdf",
				Homepage:    "https://goreleaser.com/",
				Bindir:      "/usr/bin",
				NFPMOverridables: config.NFPMOverridables{
					FileNameTemplate: `
						{{- trimsuffix .ConventionalFileName .ConventionalExtension -}}
						{{- if and (eq .Arm "6") (eq .ConventionalExtension ".deb") }}6{{ end -}}
						{{- if not (eq .Amd64 "v1")}}{{ .Amd64 }}{{ end -}}
						{{- .ConventionalExtension -}}
					`,
					PackageName: "foo{{ if .IsSnapshot }}-snapshot{{ end }}",
				},
			},
		},
	}, testctx.WithVersion("1.0.0"), testctx.WithCurrentTag("v1.0.0"))

	if snapshot {
		ctx.Snapshot = true
	}
	for _, goos := range []string{"linux", "darwin", "aix"} {
		for _, goarch := range []string{"amd64", "386", "arm64", "arm", "mips", "ppc64", "riscv64"} {
			if goarch == "ppc64" && goos != "aix" {
				continue
			}

			switch goarch {
			case "arm64":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    "subdir/mybin",
					Path:    binPath,
					Goarch:  goarch,
					Goos:    goos,
					Goarm64: "v8.0",
					Type:    artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: "default",
					},
				})
			case "arm":
				for _, goarm := range []string{"6", "7"} {
					ctx.Artifacts.Add(&artifact.Artifact{
						Name:   "subdir/mybin",
						Path:   binPath,
						Goarch: goarch,
						Goos:   goos,
						Goarm:  goarm,
						Type:   artifact.Binary,
						Extra: map[string]any{
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
						Extra: map[string]any{
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
						Extra: map[string]any{
							artifact.ExtraID: "default",
						},
					})
				}
			case "386":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "subdir/mybin",
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Go386:  "sse2",
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: "default",
					},
				})
			case "riscv64":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:      "subdir/mybin",
					Path:      binPath,
					Goarch:    goarch,
					Goos:      goos,
					Goriscv64: "rva22u64",
					Type:      artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: "default",
					},
				})
			case "ppc64":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    "subdir/mybin",
					Path:    binPath,
					Goarch:  goarch,
					Goos:    goos,
					Goppc64: "power9",
					Type:    artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: "default",
					},
				})
			default:
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
	}
	require.NoError(t, Pipe{}.Run(ctx))
	packages := ctx.Artifacts.Filter(artifact.ByType(artifact.LinuxPackage)).List()
	require.Len(t, packages, 52)
	prefix := "foo"
	if snapshot {
		prefix += "-snapshot"
	}
	for _, pkg := range packages {
		format := pkg.Format()
		require.NotEmpty(t, format)
		require.Contains(t, []string{
			prefix + "-1.0.0-1.aarch64.rpm",
			prefix + "-1.0.0-1.armv6hl.rpm",
			prefix + "-1.0.0-1.armv7hl.rpm",
			prefix + "-1.0.0-1.i386.rpm",
			prefix + "-1.0.0-1.mips.rpm",
			prefix + "-1.0.0-1.mips.rpm",
			prefix + "-1.0.0-1.x86_64.rpm",
			prefix + "-1.0.0-1.x86_64v2.rpm",
			prefix + "-1.0.0-1.x86_64v3.rpm",
			prefix + "-1.0.0-1.x86_64v4.rpm",
			prefix + "-1.0.0-1.ppc.rpm",
			prefix + "_1.0.0_aarch64.apk",
			prefix + "_1.0.0_amd64.deb",
			prefix + "_1.0.0_amd64.ipk",
			prefix + "_1.0.0_amd64v2.deb",
			prefix + "_1.0.0_amd64v2.ipk",
			prefix + "_1.0.0_amd64v3.deb",
			prefix + "_1.0.0_amd64v3.ipk",
			prefix + "_1.0.0_amd64v4.deb",
			prefix + "_1.0.0_amd64v4.ipk",
			prefix + "_1.0.0_arm64.deb",
			prefix + "_1.0.0_arm64.ipk",
			prefix + "_1.0.0_armhf.apk",
			prefix + "_1.0.0_armhf.deb",
			prefix + "_1.0.0_armhf.ipk",
			prefix + "_1.0.0_armhf6.deb",
			prefix + "_1.0.0_armhf6.ipk",
			prefix + "_1.0.0_armv7.apk",
			prefix + "_1.0.0_i386.deb",
			prefix + "_1.0.0_i386.ipk",
			prefix + "_1.0.0_mips.apk",
			prefix + "_1.0.0_mips.deb",
			prefix + "_1.0.0_mips.ipk",
			prefix + "_1.0.0_mips.apk",
			prefix + "_1.0.0_mips.deb",
			prefix + "_1.0.0_mips.ipk",
			prefix + "_1.0.0_x86.apk",
			prefix + "_1.0.0_x86.ipk",
			prefix + "_1.0.0_x86_64.apk",
			prefix + "_1.0.0_x86_64.ipk",
			prefix + "_1.0.0_x86_64v2.apk",
			prefix + "_1.0.0_x86_64v2.ipk",
			prefix + "_1.0.0_x86_64v3.apk",
			prefix + "_1.0.0_x86_64v3.ipk",
			prefix + "_1.0.0_x86_64v4.apk",
			prefix + "_1.0.0_x86_64v4.ipk",
			prefix + "-1.0.0-1-aarch64.pkg.tar.zst",
			prefix + "-1.0.0-1-armv6h.pkg.tar.zst",
			prefix + "-1.0.0-1-armv7h.pkg.tar.zst",
			prefix + "-1.0.0-1-i686.pkg.tar.zst",
			prefix + "-1.0.0-1-x86_64.pkg.tar.zst",
			prefix + "-1.0.0-1-x86_64v2.pkg.tar.zst",
			prefix + "-1.0.0-1-x86_64v3.pkg.tar.zst",
			prefix + "-1.0.0-1-x86_64v4.pkg.tar.zst",
			prefix + "-1.0.0-1-mips.pkg.tar.zst",
			prefix + "-1.0.0-1-mips.pkg.tar.zst",
			prefix + "-1.0.0-1-riscv64.pkg.tar.zst",
			prefix + "_1.0.0_riscv64.apk",
			prefix + "-1.0.0-1.riscv64.rpm",
			prefix + "_1.0.0_riscv64.ipk",
			prefix + "_1.0.0_riscv64.deb",
		}, pkg.Name, "package name is not expected")
		require.Equal(t, "someid", pkg.ID())
		require.ElementsMatch(t, []string{binPath}, sources(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
		require.ElementsMatch(t, []string{"/usr/bin/subdir/mybin"}, destinations(artifact.MustExtra[files.Contents](*pkg, extraFiles)))
	}
}

func TestIntegrationDebSpecificConfig(t *testing.T) {
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
						Contents: []config.NFPMContent{
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
		return ctx
	}

	t.Run("no passphrase set", func(t *testing.T) {
		require.Contains(
			t,
			Pipe{}.Run(setupContext(t)).Error(),
			`key is encrypted but no passphrase was provided`,
		)
	})

	t.Run("global passphrase set", func(t *testing.T) {
		ctx := setupContext(t)
		ctx.Env = map[string]string{
			"NFPM_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
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
		ctx.Parallelism = 100
		ctx.Env = map[string]string{
			"NFPM_SOMEID_DEB_PASSPHRASE": "hunter2",
		}
		ctx.Config.NFPMs[0].Deb.Lintian = []string{
			"statically-linked-binary",
			"changelog-file-missing-in-native-package",
		}
		ctx.Config.NFPMs[0].Formats = []string{"apk", "rpm", "deb", "termux.deb", "ipk"}

		require.NoError(t, Pipe{}.Run(ctx))
		for _, format := range []string{"apk", "rpm", "ipk"} {
			require.NoDirExists(t, filepath.Join(ctx.Config.Dist, format))
		}
		require.DirExists(t, filepath.Join(ctx.Config.Dist, "deb"))
		bts, err := os.ReadFile(filepath.Join(ctx.Config.Dist, "deb", "foo_amd64", "lintian"))
		require.NoError(t, err)
		require.Equal(t, "foo: statically-linked-binary\nfoo: changelog-file-missing-in-native-package", string(bts))
	})

	t.Run("lintian no debs", func(t *testing.T) {
		ctx := setupContext(t)
		ctx.Parallelism = 100
		ctx.Env = map[string]string{
			"NFPM_SOMEID_DEB_PASSPHRASE": "hunter2",
		}
		ctx.Config.NFPMs[0].Deb.Lintian = []string{
			"statically-linked-binary",
			"changelog-file-missing-in-native-package",
		}
		ctx.Config.NFPMs[0].Formats = []string{"apk", "rpm", "ipk"}

		require.NoError(t, Pipe{}.Run(ctx))
		for _, format := range []string{"deb", "termux.deb"} {
			require.NoDirExists(t, filepath.Join(ctx.Config.Dist, format))
		}
	})
}

func TestIntegrationRPMSpecificConfig(t *testing.T) {
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
					Contents: []config.NFPMContent{
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
			"NFPM_SOMEID_RPM_PASSPHRASE": "hunter2",
		}
		require.NoError(t, Pipe{}.Run(ctx))
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
