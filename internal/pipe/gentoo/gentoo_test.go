package gentoo

import (
	"bytes"
	"os"
	"path/filepath"
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
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
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
		{Content: []byte("<pkgmetadata></pkgmetadata>"), Path: "app-misc/foo/metadata.xml"},
	}

	err := handleGentooManifestAndMetadata(ctx, cfg, nil, client.Repo{}, &files, nil)
	require.NoError(t, err)

	manifestIdx := len(files) - 1
	manifestContent := string(files[manifestIdx].Content)
	require.Contains(t, manifestContent, "DIST foo_1.0.0_linux_amd64.tar.gz")
	require.Contains(t, manifestContent, "EBUILD foo-1.0.0.ebuild")
	require.Contains(t, manifestContent, "AUX foo.patch")
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
	content []byte
}

func (m mockFileDownloader) DownloadFile(_ *import_context.Context, _ client.Repo, path string) ([]byte, error) {
	if path == "metadata/layout.conf" {
		return m.content, nil
	}
	return nil, client.ErrNotFound
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
