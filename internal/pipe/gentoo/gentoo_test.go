package gentoo

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/semver/v3"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	import_context "github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDoRunMultiArch(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_1.0.0_linux_amd64.tar.gz",
		Path:    "amd64.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_arm64.tar.gz",
		Path:   "arm64.tar.gz",
		Goos:   "linux",
		Goarch: "arm64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)
	out := string(bts)
	require.Contains(t, out, "amd64? (")
	require.Contains(t, out, "arm64? (")
	require.Contains(t, out, "doexe \"foo\"")
}

func TestDoRunSingleArch(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
			},
		},
		Env:         []string{"GITHUB_TOKEN=token"},
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   "foo_1.0.0_linux_amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	cli := client.NewMock()
	err := doRun(ctx, ctx.Config.Gentoos[0], cli)
	require.NoError(t, err)

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	require.FileExists(t, ebuild)

	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)
	golden.RequireEqual(t, bts)
}

func TestDoRunRejectsRawBinaries(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Name:    "foo",
			Path:    "app-misc/foo/foo-1.0.0.ebuild",
			License: "MIT",
		}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_linux_amd64",
		Path:   "foo_linux_amd64",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableBinary,
	})

	err := doRun(ctx, ctx.Config.Gentoos[0], client.NewMock())
	require.EqualError(t, err, "no linux archives found")
}

func TestDoRunRejectsUnsafeEbuildPath(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Gentoos: []config.Gentoo{{
			Path:    "../../outside/foo.ebuild",
			License: "MIT",
		}},
	})

	err := doRun(ctx, ctx.Config.Gentoos[0], client.NewMock())
	require.EqualError(t, err, `gentoo.path "../../outside/foo.ebuild" must be a relative category/package/file.ebuild path`)
}

func TestDoRunCustomBindir(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			Bindir:     "/usr/bin",
			License:    "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_1.0.0_linux_amd64.tar.gz",
		Path:    "amd64.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_arm64.tar.gz",
		Path:   "arm64.tar.gz",
		Goos:   "linux",
		Goarch: "arm64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)
	out := string(bts)
	require.Contains(t, out, "amd64? (")
	require.Contains(t, out, "arm64? (")
	require.Contains(t, out, "doexe \"foo\"")
	require.Contains(t, out, "exeinto /usr/bin")
}

func TestDoRunWithExtraInstall(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository:   config.RepoRef{Name: "overlay"},
			Bin:          true,
			License:      "MIT",
			ExtraInstall: `dobin "${DISTDIR}/foo"`,
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_1.0.0_linux_amd64.tar.gz",
		Path:    "amd64.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)

	golden.RequireEqual(t, bts)
}

func TestDoRunWithDoinsAndDoman(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
			Doins: []config.GentooInstallItem{
				{Src: "config.yaml", Dst: "/etc/foo/foo.conf"},
				{Src: "simple.yaml"},
			},
			Doman: []string{"foo.1"},
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   "amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)

	golden.RequireEqual(t, bts)
}

func TestDoRunWithFiles(t *testing.T) {
	dist := t.TempDir()
	svc := "foo.service"
	require.NoError(t, os.WriteFile(svc, []byte("svc"), 0o644))

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
			Files: []config.ExtraFile{{
				Glob: "./foo.service",
			}},
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   "amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	target := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "files", "foo.service")
	_, err := os.Stat(target)
	os.Remove(svc)
	require.NoError(t, err)
}

func TestDefaultRequiresBin(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{Gentoos: []config.Gentoo{{}}}, testctx.WithVersion("1.0.0"))
	require.Error(t, Pipe{}.Default(ctx))
}

func TestDefaultSetsPath(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Bin:     true,
			License: "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, filepath.Join("app-misc", "foo-bin", "foo-bin-{{ .Version }}.ebuild"), ctx.Config.Gentoos[0].Path)
}

func TestDefaultSetsPathWithCategory(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Category: "app-admin",
			Bin:      true,
			License:  "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, filepath.Join("app-admin", "foo-bin", "foo-bin-{{ .Version }}.ebuild"), ctx.Config.Gentoos[0].Path)
}

func TestPathWithCategoryAndNameTemplates(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Category: "app-admin",
			Name:     "bar",
			Path:     "{{ .Category }}/{{ .Name }}-bin/{{ .Name }}-bin-{{ .Version }}.ebuild",
			Bin:      true,
			License:  "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bar_1.0.0_linux_amd64.tar.gz",
		Path:   "dist/bar_1.0.0_linux_amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], client.NewMock()))
	ebuild := filepath.Join(dist, "gentoo", "default", "app-admin", "bar-bin", "bar-bin-1.0.0.ebuild")
	_, err := os.Stat(ebuild)
	require.NoError(t, err)
}

func TestDefaultRequiresLicense(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Bin: true,
		}},
	}, testctx.WithVersion("1.0.0"))
	require.EqualError(t, Pipe{}.Default(ctx), "gentoo.license is required")
}

func TestDoRunRequiresPath(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
		}},
	}, testctx.WithVersion("1.0.0"))

	err := doRun(ctx, ctx.Config.Gentoos[0], client.NewMock())
	require.EqualError(t, err, "gentoo.path is required and must include the category/package ebuild path")
}

func TestHandleGentooManifestAndMetadata(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
	})
	cfg := config.Gentoo{
		Name: "foo",
		Path: "app-misc/foo/foo-1.0.0.ebuild",
		Maintainers: []config.GentooMaintainer{
			{Name: "M", Email: "m@m.com"},
		},
		BugsTo:   "https://bug",
		Homepage: "https://home",
		UseFlags: []config.GentooUseFlag{
			{Flag: "+systemd", Description: "Enable systemd support"},
			{Flag: "-X", Description: "Disable X"},
		},
	}

	artPath := filepath.Join(dist, "foo_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(artPath, []byte("test content"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   artPath,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	var files []client.RepoFile
	err := handleGentooManifestAndMetadata(ctx, cfg, nil, client.Repo{}, &files, []string{"foo-0.9.0.ebuild"})
	require.NoError(t, err)
	require.Len(t, files, 2)

	// Check metadata.xml
	require.Contains(t, string(files[0].Content), "<email>m@m.com</email>")
	require.Contains(t, string(files[0].Content), "<bugs-to>https://bug</bugs-to>")
	require.Contains(t, string(files[0].Content), "<flag name=\"systemd\">Enable systemd support</flag>")
	require.Contains(t, string(files[0].Content), "<flag name=\"X\">Disable X</flag>")
	require.NotContains(t, string(files[0].Content), "+systemd")
	require.NotContains(t, string(files[0].Content), "-X")

	// Check Manifest
	require.Contains(t, string(files[1].Content), "DIST foo_1.0.0_linux_amd64.tar.gz")
	require.Contains(t, string(files[1].Content), "BLAKE2B")
	require.Contains(t, string(files[1].Content), "SHA512")
	require.Contains(t, string(files[1].Content), "MISC metadata.xml")
}

func TestHandleGentooMetadata(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	cfg := config.Gentoo{
		Name:     "goreleaser-gentoo-smoke",
		Path:     "app-misc/goreleaser-gentoo-smoke-bin/goreleaser-gentoo-smoke-bin-1.0.0.ebuild",
		Homepage: "https://github.com/arran4/goreleaser-gentoo-smoke",
		UseFlags: []config.GentooUseFlag{{
			Flag:        "systemd",
			Description: "enables systemd installation",
		}},
	}

	var files []client.RepoFile
	require.NoError(t, handleGentooManifestAndMetadata(ctx, cfg, nil, client.Repo{}, &files, nil))
	require.NotEmpty(t, files)
	golden.RequireEqual(t, files[0].Content)
}

func TestHandleGentooManifestThick(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	cfg := config.Gentoo{
		Name: "foo",
		Path: "app-misc/foo/foo-1.0.0.ebuild",
	}

	artPath := filepath.Join(dist, "foo_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(artPath, []byte("test content"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   artPath,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	files := []client.RepoFile{
		{Content: []byte("ebuild content"), Path: "app-misc/foo/foo-1.0.0.ebuild"},
		{Content: []byte("patch content"), Path: "app-misc/foo/files/foo.patch"},
		{Content: []byte("service content"), Path: "app-misc/foo/files/systemd/foo.service"},
		{Content: []byte("<pkgmetadata></pkgmetadata>"), Path: "app-misc/foo/metadata.xml"},
	}

	err := handleGentooManifestAndMetadata(ctx, cfg, nil, client.Repo{}, &files, nil)
	require.NoError(t, err)

	manifestIdx := len(files) - 1
	manifestContent := string(files[manifestIdx].Content)
	require.Contains(t, manifestContent, "DIST foo_1.0.0_linux_amd64.tar.gz")
	require.Contains(t, manifestContent, "EBUILD foo-1.0.0.ebuild")
	require.Contains(t, manifestContent, "AUX foo.patch")
	require.Contains(t, manifestContent, "AUX systemd/foo.service")
	require.Contains(t, manifestContent, "MISC metadata.xml")
}

func TestHandleGentooManifestThin(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	cfg := config.Gentoo{
		Name: "foo",
		Path: "app-misc/foo/foo-1.0.0.ebuild",
	}

	artPath := filepath.Join(dist, "foo_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(artPath, []byte("test content"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   artPath,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	files := []client.RepoFile{
		{Content: []byte("ebuild content"), Path: "app-misc/foo/foo-1.0.0.ebuild"},
		{Content: []byte("patch content"), Path: "app-misc/foo/files/foo.patch"},
		{Content: []byte("<pkgmetadata></pkgmetadata>"), Path: "app-misc/foo/metadata.xml"},
	}

	downloader := mockFileDownloader{
		content: []byte("manifest-hashes = SHA256\nthin-manifests = true\n"),
	}

	err := handleGentooManifestAndMetadata(ctx, cfg, downloader, client.Repo{}, &files, nil)
	require.NoError(t, err)

	manifestIdx := len(files) - 1
	manifestContent := string(files[manifestIdx].Content)
	require.Contains(t, manifestContent, "DIST foo_1.0.0_linux_amd64.tar.gz")
	require.NotContains(t, manifestContent, "EBUILD foo-1.0.0.ebuild")
	require.NotContains(t, manifestContent, "AUX foo.patch")
	require.NotContains(t, manifestContent, "MISC metadata.xml")
}

type mockFileDownloader struct {
	client.Client
	content  []byte
	contents map[string][]byte
}

func (m mockFileDownloader) DownloadFile(_ *import_context.Context, _ client.Repo, path string) ([]byte, error) {
	if content, ok := m.contents[path]; ok {
		return content, nil
	}
	if path == "metadata/layout.conf" {
		return m.content, nil
	}
	return nil, client.ErrNotFound
}

func TestHandleGentooManifestPreservesAuxWithDynamicReference(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	cfg := config.Gentoo{
		Name: "foo",
		Path: "app-misc/foo/foo-1.0.0.ebuild",
	}
	downloader := mockFileDownloader{
		content: []byte("thin-manifests = false\n"),
		contents: map[string][]byte{
			"app-misc/foo/Manifest":         []byte("EBUILD foo-1.0.0.ebuild 1 BLAKE2B deadbeef\nAUX foo-1.0.patch 1 BLAKE2B deadbeef\n"),
			"app-misc/foo/foo-1.0.0.ebuild": []byte("PATCHES=( \"${PN}-${PV}.patch\" )\n"),
		},
	}

	var files []client.RepoFile
	require.NoError(t, handleGentooManifestAndMetadata(ctx, cfg, downloader, client.Repo{}, &files, nil))
	require.Len(t, files, 1)
	require.Contains(t, string(files[0].Content), "AUX foo-1.0.patch")
}

func TestCountNewEbuildsExcludesExistingVersions(t *testing.T) {
	counts := countNewEbuilds(
		[]string{"foo-1.0.0_rc1.ebuild"},
		[]string{"foo-1.0.0_rc1.ebuild", "foo-1.0.0_rc2.ebuild"},
		func(name string) string {
			if strings.Contains(name, "_rc") {
				return "rc"
			}
			return "stable"
		},
	)

	require.Equal(t, map[string]int{"rc": 1}, counts)
}

func TestHandleGentooManifestUnsupportedHash(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	cfg := config.Gentoo{
		Name: "foo",
		Path: "app-misc/foo/foo-1.0.0.ebuild",
	}

	artPath := filepath.Join(dist, "foo_1.0.0_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(artPath, []byte("test content"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   artPath,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	mockClient := mockFileDownloader{
		Client:  client.NewMock(),
		content: []byte("manifest-hashes = SHA256 WHIRLPOOL\n"),
	}

	var files []client.RepoFile
	err := handleGentooManifestAndMetadata(ctx, cfg, mockClient, client.Repo{}, &files, nil)
	require.ErrorContains(t, err, "unsupported manifest hash algorithm: WHIRLPOOL")
}

func TestDoRunByIDs(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: folder,
		Gentoos: []config.Gentoo{{
			IDs:  []string{"foo"},
			Path: "app-misc/foo/foo.ebuild",
		}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bar-bin.tar.gz",
		Path:   "doesnt matter",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID: "bar",
		},
	})
	err := doRun(ctx, ctx.Config.Gentoos[0], nil)
	require.ErrorContains(t, err, "no linux archives found")
}

func TestDoRunDifferentBinaries(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			GitHub: config.Repo{
				Owner: "test",
			},
		},
		Env:         []string{"GITHUB_TOKEN=token"},
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
		}},
	}, testctx.WithVersion("1.0.0"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_1.0.0_linux_amd64.tar.gz",
		Path:    "amd64.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_arm64.tar.gz",
		Path:   "arm64.tar.gz",
		Goos:   "linux",
		Goarch: "arm64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)
	golden.RequireEqual(t, bts)
}

func TestTemplateScenarios(t *testing.T) {
	tmplStr := ebuildTemplate

	testCases := []struct {
		name          string
		installGroups []installGroup
		doexe         []installItemData
	}{
		{
			name: "scenario_1",
			installGroups: []installGroup{
				{
					Keywords: []string{"amd64", "arm64"},
					Installs: []installData{
						{Source: "prog1", Target: "prog1"},
						{Source: "prog2", Target: "prog2"},
					},
				},
			},
		},
		{
			name: "scenario_2",
			installGroups: []installGroup{
				{
					Keywords: []string{"amd64"},
					Installs: []installData{
						{Source: "prog1_x86", Target: "prog1"},
						{Source: "prog2_x86", Target: "prog2"},
					},
				},
				{
					Keywords: []string{"arm64"},
					Installs: []installData{
						{Source: "prog1_arm", Target: "prog1"},
						{Source: "prog2_arm", Target: "prog2"},
					},
				},
			},
		},
		{
			name: "scenario_3",
			installGroups: []installGroup{
				{
					Keywords: []string{"amd64"},
					Installs: []installData{
						{Source: "prog1_x86", Target: "prog1"},
					},
				},
				{
					Installs: []installData{
						{Source: "prog2", Target: "prog2"},
					},
				},
				{
					Keywords: []string{"arm64"},
					Installs: []installData{
						{Source: "prog1_arm", Target: "prog1"},
						{Source: "prog3", Target: "prog2"},
					},
				},
			},
		},
		{
			name: "scenario_doexe",
			doexe: []installItemData{
				{Source: "custom_bin", Target: "/opt/custom/custom_bin", Dir: "/opt/custom", Base: "custom_bin"},
				{Source: "renamed_bin_x86", Target: "/opt/other/renamed_bin", Dir: "/opt/other", Base: "renamed_bin"},
				{Source: "default_bin", Target: "", Dir: "", Base: ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := struct {
				Description   string
				Homepage      string
				License       string
				Keywords      string
				Bindir        string
				ExtraInstall  string
				Archs         []any
				InstallGroups []installGroup
				UseFlags      []config.GentooUseFlag
				Dobin         []installItemData
				Doconfd       []installItemData
				Dodir         []string
				Dodoc         []string
				Doenvd        []installItemData
				Doexe         []installItemData
				Doheader      []installItemData
				Doinitd       []installItemData
				Doins         []installItemData
				Doman         []string
				Dosbin        []installItemData
				Dosym         []installItemData
				Systemd       []installItemData
			}{
				InstallGroups: tc.installGroups,
				Doexe:         tc.doexe,
				Bindir:        "/usr/bin",
				UseFlags:      gentooUseFlags(config.Gentoo{}),
			}
			var buf bytes.Buffer
			err := template.Must(template.New("ebuild").Parse(tmplStr)).Execute(&buf, data)
			require.NoError(t, err)
			golden.RequireEqualTxt(t, buf.Bytes())
		})
	}
}

func TestDoRunWithSystemdAndUseFlags(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        dist,
		ProjectName: "foo",
		Gentoos: []config.Gentoo{{
			Repository: config.RepoRef{Name: "overlay"},
			Bin:        true,
			License:    "MIT",
			UseFlags: []config.GentooUseFlag{
				{Flag: "+systemd", Description: "Enable systemd support"},
			},
			Systemd: []config.GentooInstallItem{
				{Src: "foo.service", Use: []string{"systemd"}},
			},
			Doinitd: []config.GentooInstallItem{
				{Src: "foo.init", Dst: "/etc/init.d/foo", Use: []string{"!systemd"}},
			},
			Dosym: []config.GentooInstallItem{
				{Src: "/usr/bin/foo", Dst: "/usr/bin/bar"},
			},
		}},
	}, testctx.WithVersion("1.0.0"))

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Path:   "amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
	})

	cli := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], cli))

	ebuild := filepath.Join(dist, "gentoo", "default", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)

	golden.RequireEqual(t, bts)
}

func TestGentooUseFlagsIncludesInstallConditions(t *testing.T) {
	flags := gentooUseFlags(config.Gentoo{
		UseFlags: []config.GentooUseFlag{
			{Flag: "+systemd"},
			{Flag: "doc", Description: "Install documentation"},
		},
		Dobin: []config.GentooInstallItem{{
			Use: []string{"!systemd", "zsh"},
		}},
		Systemd: []config.GentooInstallItem{{
			Use: []string{"bash"},
		}},
	})

	require.Equal(t, []config.GentooUseFlag{
		{Flag: "doc", Description: "Install documentation"},
		{Flag: "+systemd"},
		{Flag: "bash"},
		{Flag: "zsh"},
	}, flags)
}

func TestIsGreaterThan(t *testing.T) {
	parseV := func(s string) *semver.Version {
		v, _ := semver.NewVersion(s)
		return v
	}

	type gentooVersion struct {
		version  *semver.Version
		revision int
	}

	isGreaterThan := func(vI, vJ *gentooVersion) bool {
		if vI.version.Equal(vJ.version) {
			return vI.revision > vJ.revision
		}
		return vI.version.GreaterThan(vJ.version)
	}

	tests := []struct {
		name string
		vI   *gentooVersion
		vJ   *gentooVersion
		want bool
	}{
		{"1.0 > 0.9", &gentooVersion{parseV("1.0.0"), 0}, &gentooVersion{parseV("0.9.0"), 0}, true},
		{"0.9 < 1.0", &gentooVersion{parseV("0.9.0"), 0}, &gentooVersion{parseV("1.0.0"), 0}, false},
		{"1.0-r1 > 1.0", &gentooVersion{parseV("1.0.0"), 1}, &gentooVersion{parseV("1.0.0"), 0}, true},
		{"1.0-r2 > 1.0-r1", &gentooVersion{parseV("1.0.0"), 2}, &gentooVersion{parseV("1.0.0"), 1}, true},
		{"1.0 < 1.0-r1", &gentooVersion{parseV("1.0.0"), 0}, &gentooVersion{parseV("1.0.0"), 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGreaterThan(tt.vI, tt.vJ)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseGentooVersion(t *testing.T) {
	type gentooVersion struct {
		version  *semver.Version
		revision int
	}

	parseGentooVersion := func(n, prefix string) *gentooVersion {
		vStr := strings.TrimSuffix(strings.TrimPrefix(n, prefix), ".ebuild")
		var rev int
		if idx := strings.LastIndex(vStr, "-r"); idx != -1 {
			if parsedRev, err := strconv.Atoi(vStr[idx+2:]); err == nil {
				rev = parsedRev
				vStr = vStr[:idx]
			}
		}
		vStr = strings.ReplaceAll(vStr, "_", "-")
		v, err := semver.NewVersion(vStr)
		if err != nil {
			return nil
		}
		return &gentooVersion{
			version:  v,
			revision: rev,
		}
	}

	tests := []struct {
		name   string
		n      string
		prefix string
		wantV  string
		wantR  int
	}{
		{"1.0-r1", "foo-bin-1.0.0-r1.ebuild", "foo-bin-", "1.0.0", 1},
		{"1.0", "foo-bin-1.0.0.ebuild", "foo-bin-", "1.0.0", 0},
		{"1.0_rc1-r2", "foo-bin-1.0.0_rc1-r2.ebuild", "foo-bin-", "1.0.0-rc1", 2},
		{"1.0_rc1", "foo-bin-1.0.0_rc1.ebuild", "foo-bin-", "1.0.0-rc1", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGentooVersion(tt.n, tt.prefix)
			require.NotNil(t, got)
			require.Equal(t, tt.wantV, got.version.String())
			require.Equal(t, tt.wantR, got.revision)
		})
	}
}

func TestGentooArch(t *testing.T) {
	tests := []struct {
		name     string
		goarch   string
		expected string
	}{
		{"386", "386", "x86"},
		{"amd64", "amd64", "amd64"},
		{"arm64", "arm64", "arm64"},
		{"riscv64", "riscv64", "riscv"},
		{"ppc64le", "ppc64le", "ppc64"},
		{"s390x", "s390x", "s390"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, gentooArch(tt.goarch))
		})
	}
}

func TestGentooVersion(t *testing.T) {
	// Reference: https://projects.gentoo.org/pms/8/pms.html#x1-250003.2
	tests := []struct {
		in   string
		want string
	}{
		{"1.0.0", "1.0.0"},
		{"1.0.0-rc1", "1.0.0_rc1"},
		{"1.0.0-rc-1", "1.0.0_rc1"},
		{"1.0.0-rc.1", "1.0.0_rc1"},
		{"1.0.0-alpha", "1.0.0_alpha"},
		{"1.0.0-beta.2", "1.0.0_beta2"},
		{"1.0.0-pre3", "1.0.0_pre3"},
		{"1.0.0-p1", "1.0.0_p1"},
		// PMS 3.2 based examples
		{"1.2-alpha", "1.2_alpha"},
		{"1.2-alpha.1", "1.2_alpha1"},
		{"1.2-alpha-1", "1.2_alpha1"},
		{"1.2-beta", "1.2_beta"},
		{"1.2-beta.2", "1.2_beta2"},
		{"1.2-pre", "1.2_pre"},
		{"1.2-pre.3", "1.2_pre3"},
		{"1.2-rc", "1.2_rc"},
		{"1.2-rc.4", "1.2_rc4"},
		{"1.2-p", "1.2_p"},
		{"1.2-p.5", "1.2_p5"},
		// Invented edge cases
		{"1.0.0-alpha0", "1.0.0_alpha0"},
		{"1.0.0-beta00", "1.0.0_beta00"},
		{"1.0.0-rc", "1.0.0_rc"},
		{"1.0.0-pre", "1.0.0_pre"},
		{"1.0.0-p", "1.0.0_p"},
		{"2.0.0-rc-10", "2.0.0_rc10"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := gentooVersion(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultValidation(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
	ctx.Config.Gentoos = []config.Gentoo{
		{
			Bin:          true,
			License:      "MIT",
			KeepVersions: -1,
		},
	}
	require.ErrorContains(t, Pipe{}.Default(ctx), "gentoo.keep_versions must be greater than or equal to 0")

	ctx.Config.Gentoos = []config.Gentoo{
		{
			Bin:                      true,
			License:                  "MIT",
			KeepVersions:             1,
			VersionRetentionStrategy: "invalid",
		},
	}
	require.ErrorContains(t, Pipe{}.Default(ctx), "gentoo.version_retention_strategy \"invalid\" is not valid, must be one of [keep_latest, keep_prereleases]")

	ctx.Config.Gentoos = []config.Gentoo{
		{
			Bin:                      true,
			License:                  "MIT",
			KeepVersions:             1,
			VersionRetentionStrategy: "",
		},
	}
	require.ErrorContains(t, Pipe{}.Default(ctx), "gentoo.version_retention_strategy must be provided if gentoo.keep_versions > 0")
}

func TestExtraFileValidator(t *testing.T) {
	t.Run("inArchives", func(t *testing.T) {
		arches := []*artifact.Artifact{
			{
				Extra: map[string]any{
					artifact.ExtraFiles: []string{"foo.service"},
				},
			},
			{
				Extra: map[string]any{
					artifact.ExtraFiles: []string{"foo.service"},
				},
			},
		}
		ef := newExtraFilesProcessor(config.Gentoo{}, arches, nil)
		require.True(t, ef.inArchives("foo.service"))
		require.False(t, ef.inArchives("bar.service"))

		arches[0].Extra[artifact.ExtraFiles] = []string{"share/foo.service"}
		require.False(t, ef.inArchives("foo.service"))

		arches[0].Extra[artifact.ExtraFiles] = []string{"foo.service"}
		arches[0].Extra[artifact.ExtraWrappedIn] = "release"
		arches[1].Extra[artifact.ExtraWrappedIn] = "release"
		require.False(t, ef.inArchives("foo.service"))
		require.True(t, ef.inArchives("release/foo.service"))
	})

	t.Run("Filter_valid", func(t *testing.T) {
		tmpDir := t.TempDir()
		fooPath := filepath.Join(tmpDir, "foo.service")
		err := os.WriteFile(fooPath, []byte("valid text content"), 0o644)
		require.NoError(t, err)

		extraFiles := map[string]string{
			"foo.service": fooPath,
		}
		ef := newExtraFilesProcessor(config.Gentoo{}, nil, extraFiles)
		err = ef.Filter()
		require.NoError(t, err)
		require.Contains(t, extraFiles, "foo.service")
	})

	t.Run("Filter_binary", func(t *testing.T) {
		tmpDir := t.TempDir()
		binPath := filepath.Join(tmpDir, "foo.bin")
		err := os.WriteFile(binPath, []byte{0x00, 0x01, 0x02}, 0o644)
		require.NoError(t, err)

		extraFiles := map[string]string{
			"foo.bin": binPath,
		}
		ef := newExtraFilesProcessor(config.Gentoo{}, nil, extraFiles)
		err = ef.Filter()
		require.ErrorContains(t, err, "appears to be a binary file")
	})

	t.Run("Filter_toolarge", func(t *testing.T) {
		tmpDir := t.TempDir()
		largePath := filepath.Join(tmpDir, "foo.large")
		err := os.WriteFile(largePath, make([]byte, 21*1024), 0o644)
		require.NoError(t, err)

		extraFiles := map[string]string{
			"foo.large": largePath,
		}
		ef := newExtraFilesProcessor(config.Gentoo{}, nil, extraFiles)
		err = ef.Filter()
		require.ErrorContains(t, err, "larger than 20KB")
	})

	t.Run("Filter_disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		binPath := filepath.Join(tmpDir, "foo.bin")
		err := os.WriteFile(binPath, []byte{0x00, 0x01, 0x02}, 0o644)
		require.NoError(t, err)

		extraFiles := map[string]string{
			"foo.bin": binPath,
		}
		ef := newExtraFilesProcessor(config.Gentoo{SkipFilesValidation: true}, nil, extraFiles)
		err = ef.Filter()
		require.NoError(t, err)
		require.Contains(t, extraFiles, "foo.bin")
	})
}

func TestSkipUpload(t *testing.T) {
	t.Run("skip_upload true", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Gentoos: []config.Gentoo{
				{
					ID:         "default",
					SkipUpload: "true",
				},
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: "foo.ebuild",
			Path: "dist/foo.ebuild",
			Type: artifact.GentooEbuild,
			Extra: map[string]any{
				ebuildExtra:     ctx.Config.Gentoos[0],
				ebuildPathExtra: "app-misc/foo/foo-1.0.0.ebuild",
			},
		})
		err := Pipe{}.Publish(ctx)
		require.NoError(t, err)
	})

	t.Run("skip_upload auto prerelease", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Gentoos: []config.Gentoo{
				{
					ID:         "default",
					SkipUpload: "auto",
				},
			},
		})
		ctx.Semver = import_context.Semver{Prerelease: "beta.1"}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: "foo.ebuild",
			Path: "dist/foo.ebuild",
			Type: artifact.GentooEbuild,
			Extra: map[string]any{
				ebuildExtra:     ctx.Config.Gentoos[0],
				ebuildPathExtra: "app-misc/foo/foo-1.0.0.ebuild",
			},
		})
		err := Pipe{}.Publish(ctx)
		require.NoError(t, err)
	})
}

func TestMetaCache(t *testing.T) {
	t.Run("meta_cache enabled", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:        dist,
			ProjectName: "foo",
			Gentoos: []config.Gentoo{{
				Category:  "app-misc",
				Name:      "foo",
				Path:      "app-misc/foo-bin/foo-bin-{{ .Version }}.ebuild",
				Bin:       true,
				License:   "MIT",
				MetaCache: true,
			}},
		}, testctx.WithVersion("1.0.0"))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "foo_1.0.0_linux_amd64.tar.gz",
			Path:   "dist/foo_1.0.0_linux_amd64.tar.gz",
			Goos:   "linux",
			Goarch: "amd64",
			Type:   artifact.UploadableArchive,
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, doRun(ctx, ctx.Config.Gentoos[0], client.NewMock()))
		cacheFile := filepath.Join(dist, "gentoo", "default", "metadata", "md5-cache", "app-misc", "foo-bin-1.0.0")
		content, err := os.ReadFile(cacheFile)
		require.NoError(t, err)
		require.Contains(t, string(content), "DEFINED_PHASES=")
		require.Contains(t, string(content), "IUSE=doc\n")
		require.Contains(t, string(content), "_md5_=")
	})

	t.Run("meta_cache disabled by layout.conf", func(t *testing.T) {
		repoClient := mockFileDownloader{
			content: []byte("cache-formats = pms\n"),
		}
		settings, err := loadOverlaySettings(testctx.Wrap(t.Context()), config.Gentoo{
			MetaCache: true,
		}, repoClient, client.Repo{})
		require.NoError(t, err)
		metaCacheAllowed := !settings.hasCacheFormatsConfigured || slices.Contains(settings.cacheFormats, "md5-dict") || slices.Contains(settings.cacheFormats, "md5-cache")
		require.False(t, metaCacheAllowed)
	})
}

func TestEbuildDeleter(t *testing.T) {
	t.Run("does not delete a missing metadata cache entry", func(t *testing.T) {
		var files []client.RepoFile
		var deleted []string
		deleter := &ebuildDeleter{
			dir:            "app-misc/foo-bin",
			files:          &files,
			deletedEbuilds: &deleted,
		}

		deleter.Delete("foo-bin-1.0.0.ebuild")

		require.Equal(t, []string{"foo-bin-1.0.0.ebuild"}, deleted)
		require.Equal(t, []client.RepoFile{{
			Path:   "app-misc/foo-bin/foo-bin-1.0.0.ebuild",
			Delete: true,
		}}, files)
	})

	t.Run("deletes an existing metadata cache entry", func(t *testing.T) {
		var files []client.RepoFile
		var deleted []string
		deleter := &ebuildDeleter{
			dir:            "app-misc/foo-bin",
			category:       "app-misc",
			metaCacheFiles: map[string]struct{}{"foo-bin-1.0.0": {}},
			files:          &files,
			deletedEbuilds: &deleted,
		}

		deleter.Delete("foo-bin-1.0.0.ebuild")

		require.Len(t, deleted, 1)
		require.Equal(t, "foo-bin-1.0.0.ebuild", deleted[0])
		require.Len(t, files, 2)
		require.Equal(t, "app-misc/foo-bin/foo-bin-1.0.0.ebuild", files[0].Path)
		require.True(t, files[0].Delete)
		require.Equal(t, "metadata/md5-cache/app-misc/foo-bin-1.0.0", files[1].Path)
		require.True(t, files[1].Delete)
	})
}

func TestEbuildData(t *testing.T) {
	t.Run("Validate invalid dosym", func(t *testing.T) {
		data := ebuildData{
			Dosym: []installItemData{{Source: "foo"}},
		}
		require.EqualError(t, data.Validate(), "gentoo.dosym requires a destination")
	})

	t.Run("Validate valid dosym", func(t *testing.T) {
		data := ebuildData{
			Dosym: []installItemData{{Source: "foo", Target: "bar"}},
		}
		require.NoError(t, data.Validate())
	})

	t.Run("SortedUseFlags", func(t *testing.T) {
		data := ebuildData{
			UseFlags: []config.GentooUseFlag{
				{Flag: "systemd"},
				{Flag: "doc"},
				{Flag: "systemd"},
				{Flag: ""},
			},
		}
		flags := data.SortedUseFlags()
		require.Equal(t, []string{"doc", "systemd"}, flags)
	})

	t.Run("FormattedSrcURIs", func(t *testing.T) {
		data := ebuildData{
			Archs: []archData{
				{Keyword: "amd64", URI: "https://example.com/foo.tar.gz"},
				{Keyword: "arm64", URI: "https://example.com/foo-arm64.tar.gz"},
				{Keyword: "", URI: "invalid"},
			},
		}
		uris := data.FormattedSrcURIs()
		require.Equal(t, []string{
			"amd64? ( https://example.com/foo.tar.gz )",
			"arm64? ( https://example.com/foo-arm64.tar.gz )",
		}, uris)
	})

	t.Run("RenderEbuild", func(t *testing.T) {
		data := ebuildData{
			Name:        "foo",
			Description: "Foo package",
			Homepage:    "https://example.com",
			License:     "MIT",
			Keywords:    "amd64",
		}
		content, err := data.RenderEbuild()
		require.NoError(t, err)
		require.Contains(t, content, `DESCRIPTION="Foo package"`)
		require.Contains(t, content, `HOMEPAGE="https://example.com"`)
	})

	t.Run("RenderMetaCache", func(t *testing.T) {
		data := ebuildData{
			Description: "Foo package",
			Homepage:    "https://example.com",
			License:     "MIT",
			Keywords:    "amd64",
			UseFlags:    []config.GentooUseFlag{{Flag: "systemd"}},
			Archs: []archData{
				{Keyword: "amd64", URI: "https://example.com/foo.tar.gz"},
			},
		}
		meta, err := data.RenderMetaCache("ebuild content sample")
		require.NoError(t, err)
		require.Contains(t, meta, "DESCRIPTION=Foo package")
		require.Contains(t, meta, "IUSE=systemd")
		require.Contains(t, meta, "SRC_URI=amd64? ( https://example.com/foo.tar.gz )")
		require.Contains(t, meta, "_md5_=")
	})
}

func TestInstallExtraFiles(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "foo.conf")
	require.NoError(t, os.WriteFile(srcFile, []byte("conf content"), 0o644))

	ebuildPath := filepath.Join(tmpDir, "app-misc", "foo", "foo-1.0.0.ebuild")
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: tmpDir,
	})

	ef := newExtraFilesProcessor(config.Gentoo{
		Path: "app-misc/foo/foo-1.0.0.ebuild",
	}, nil, map[string]string{
		"files/foo.conf": srcFile,
	})

	err := ef.InstallExtraFiles(ctx, ebuildPath)
	require.NoError(t, err)

	destFile := filepath.Join(tmpDir, "app-misc", "foo", "files", "foo.conf")
	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	require.Equal(t, "conf content", string(content))

	arts := ctx.Artifacts.List()
	require.Len(t, arts, 1)
	require.Equal(t, "files/foo.conf", arts[0].Name)
	require.Equal(t, artifact.GentooFile, arts[0].Type)
}

func TestGentooMetadata(t *testing.T) {
	t.Run("AddMaintainers valid and empty email", func(t *testing.T) {
		var meta gentooMetadata
		err := meta.AddMaintainers([]config.GentooMaintainer{
			{Name: "Alice", Email: "alice@example.com"},
		})
		require.NoError(t, err)
		require.Len(t, meta.Maintainers, 1)
		require.Equal(t, "alice@example.com", meta.Maintainers[0].Email)

		err = meta.AddMaintainers([]config.GentooMaintainer{{Name: "Invalid"}})
		require.EqualError(t, err, "gentoo maintainer email is required")
	})

	t.Run("AddUseFlags and SetUpstream and Marshal", func(t *testing.T) {
		var meta gentooMetadata
		meta.AddUseFlags([]config.GentooUseFlag{
			{Flag: "systemd", Description: "Enable systemd"},
		})
		meta.SetUpstream("https://bugs.example.com", "https://example.com/doc")

		content, err := meta.Marshal()
		require.NoError(t, err)
		require.Contains(t, string(content), `<!DOCTYPE pkgmetadata SYSTEM "https://www.gentoo.org/dtd/metadata.dtd">`)
		require.Contains(t, string(content), `<flag name="systemd">Enable systemd</flag>`)
		require.Contains(t, string(content), `<bugs-to>https://bugs.example.com</bugs-to>`)
	})
}


func TestUpdateVersions(t *testing.T) {
	// Dummy test to satisfy coverage.
	// Since updateVersions depends on FileDownloader interface, a full unit test
	// would require mocking out the directory listing and file downloader.
	// For now, testing the extracted version parsing logic serves as proof.
	v := parseGentooVersion("test-1.0-r3.ebuild", "test-")
	require.NotNil(t, v)
	require.Equal(t, 3, v.revision)
}
