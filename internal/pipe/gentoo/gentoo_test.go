package gentoo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
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
			Files: []config.ExtraFile{{
				Glob:         "./foo.service",
				NameTemplate: "files/foo.service",
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
