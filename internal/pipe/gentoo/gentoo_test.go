package gentoo

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
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

	ebuild := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
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

	ebuild := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
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

	ebuild := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
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

	ebuild := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
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

	target := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "files", "foo.service")
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

	// Check Manifest
	require.Contains(t, string(files[1].Content), "DIST foo_1.0.0_linux_amd64.tar.gz")
	require.Contains(t, string(files[1].Content), "BLAKE2B")
	require.Contains(t, string(files[1].Content), "SHA512")
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

	ebuild := filepath.Join(dist, "gentoo", "app-misc", "foo-bin", "foo-bin-1.0.0.ebuild")
	bts, err := os.ReadFile(ebuild)
	require.NoError(t, err)
	golden.RequireEqual(t, bts)
}

func TestTemplateScenarios(t *testing.T) {
	tmplStr := ebuildTemplate

	testCases := []struct {
		name     string
		installs []installData
		expected string
	}{
		{
			name: "Scenario 1: 2 archs, same filename",
			installs: []installData{
				{Source: "prog1", Target: "prog1"},
				{Source: "prog2", Target: "prog2"},
			},
			expected: "  doexe \"prog1\" || die \"Failed to install binary\"\n  doexe \"prog2\" || die \"Failed to install binary\"\n",
		},
		{
			name: "Scenario 2: 2 archs, different filenames",
			installs: []installData{
				{Source: "prog1_x86", Target: "prog1", Keywords: []string{"amd64"}},
				{Source: "prog2_x86", Target: "prog2", Keywords: []string{"amd64"}},
				{Source: "prog1_arm", Target: "prog1", Keywords: []string{"arm64"}},
				{Source: "prog2_arm", Target: "prog2", Keywords: []string{"arm64"}},
			},
			expected: "  if use amd64; then\n    newexe \"prog1_x86\" \"prog1\" || die \"Failed to install binary\"\n  fi\n  if use amd64; then\n    newexe \"prog2_x86\" \"prog2\" || die \"Failed to install binary\"\n  fi\n  if use arm64; then\n    newexe \"prog1_arm\" \"prog1\" || die \"Failed to install binary\"\n  fi\n  if use arm64; then\n    newexe \"prog2_arm\" \"prog2\" || die \"Failed to install binary\"\n  fi\n",
		},
		{
			name: "Scenario 3: hybrid",
			installs: []installData{
				{Source: "prog1_x86", Target: "prog1", Keywords: []string{"amd64"}},
				{Source: "prog2", Target: "prog2"},
				{Source: "prog1_arm", Target: "prog1", Keywords: []string{"arm64"}},
				{Source: "prog3", Target: "prog2", Keywords: []string{"arm64"}},
			},
			expected: "  if use amd64; then\n    newexe \"prog1_x86\" \"prog1\" || die \"Failed to install binary\"\n  fi\n  doexe \"prog2\" || die \"Failed to install binary\"\n  if use arm64; then\n    newexe \"prog1_arm\" \"prog1\" || die \"Failed to install binary\"\n  fi\n  if use arm64; then\n    newexe \"prog3\" \"prog2\" || die \"Failed to install binary\"\n  fi\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := struct {
				Description  string
				Homepage     string
				License      string
				Keywords     string
				Bindir       string
				ExtraInstall string
				Archs        []any
				Installs     []installData
			}{
				Installs: tc.installs,
			}
			var buf bytes.Buffer
			err := template.Must(template.New("ebuild").Parse(tmplStr)).Execute(&buf, data)
			require.NoError(t, err)
			out := buf.String()
			require.Contains(t, out, tc.expected)
		})
	}
}
