package apkbuild

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestShellQuote(t *testing.T) {
	require.Equal(t, `'It'"'"'s "$HOME"'`, shellQuote(`It's "$HOME"`))
}

func TestToAPKArch(t *testing.T) {
	for _, tt := range []struct {
		goarch string
		goarm  string
		want   string
	}{
		{goarch: "amd64", want: "x86_64"},
		{goarch: "386", want: "x86"},
		{goarch: "arm64", want: "aarch64"},
		{goarch: "arm", goarm: "6", want: "armhf"},
		{goarch: "arm", goarm: "7", want: "armv7"},
		{goarch: "ppc64le", want: "ppc64le"},
		{goarch: "s390x", want: "s390x"},
		{goarch: "riscv64", want: "riscv64"},
		{goarch: "darwin"},
	} {
		t.Run(tt.goarch+tt.goarm, func(t *testing.T) {
			require.Equal(t, tt.want, toAPKArch(tt.goarch, tt.goarm))
		})
	}
	t.Run("default GOARM", func(t *testing.T) {
		t.Setenv("GORELEASER_EXPERIMENTAL", "")
		require.Equal(t, "armhf", toAPKArch("arm", ""))
	})
	t.Run("experimental default GOARM", func(t *testing.T) {
		t.Setenv("GORELEASER_EXPERIMENTAL", "defaultgoarm")
		require.Equal(t, "armv7", toAPKArch("arm", ""))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		APKBuilds:   []config.APKBuild{{}},
	})
	require.NoError(t, Pipe{}.Default(ctx))

	got := ctx.Config.APKBuilds[0]
	require.Equal(t, "foo", got.Name)
	require.Equal(t, "0", got.Rel)
	require.Equal(t, "v1", got.Goamd64)
	require.Equal(t, []string{"!check"}, got.Options)
	require.Equal(t, defaultCommitMsg, got.CommitMessageTemplate)
	require.Equal(t, "goreleaserbot", got.CommitAuthor.Name)
	require.Equal(t, "bot@goreleaser.com", got.CommitAuthor.Email)
}

func TestSkip(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			APKBuilds: []config.APKBuild{{}},
		}, testctx.Skip(skips.APKBuild))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("configured", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			APKBuilds: []config.APKBuild{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunAllArchitectures(t *testing.T) {
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        dist,
			ProjectName: "foo",
			APKBuilds: []config.APKBuild{{
				IDs:         []string{"foo"},
				Description: `It's "$HOME"`,
				Homepage:    "https://example.com",
				License:     "MIT",
				Depends:     []string{"ca-certificates"},
				MakeDepends: []string{"tar"},
				Provides:    []string{"foo-cli"},
				Replaces:    []string{"old-foo"},
				URLTemplate: "https://example.com/releases/{{ .ArtifactName }}",
			}},
		},
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithVersion("1.2.3"),
	)

	architectures := []struct {
		goarch string
		goarm  string
		goamd  string
		alpine string
	}{
		{goarch: "amd64", goamd: "v1", alpine: "x86_64"},
		{goarch: "386", alpine: "x86"},
		{goarch: "arm64", alpine: "aarch64"},
		{goarch: "arm", goarm: "6", alpine: "armhf"},
		{goarch: "arm", goarm: "7", alpine: "armv7"},
		{goarch: "ppc64le", alpine: "ppc64le"},
		{goarch: "s390x", alpine: "s390x"},
		{goarch: "riscv64", alpine: "riscv64"},
	}
	for _, arch := range architectures {
		name := fmt.Sprintf("foo_%s%s.tar.gz", arch.goarch, arch.goarm)
		addArtifact(t, ctx, artifact.Artifact{
			Name:    name,
			Goos:    "linux",
			Goarch:  arch.goarch,
			Goarm:   arch.goarm,
			Goamd64: arch.goamd,
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:       "foo",
				artifact.ExtraFormat:   "tar.gz",
				artifact.ExtraBinaries: []string{"foo"},
			},
		})
	}
	addArtifact(t, ctx, artifact.Artifact{
		Name:    "ignored_v3.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v3",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client.NewMock()))

	file := filepath.Join(dist, "apkbuild", "foo.apkbuild")
	require.FileExists(t, file)
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Contains(t, string(content), `pkgdesc='It'"'"'s "$HOME"'`)
	require.Contains(t, string(content), `arch='aarch64 armhf armv7 ppc64le riscv64 s390x x86 x86_64'`)
	require.Contains(t, string(content), `options='!check' # prebuilt binaries`)
	require.Contains(t, string(content), `install -Dm755 "$srcdir/foo" "$pkgdir/usr/bin/foo"`)
	require.NotContains(t, string(content), "ignored_v3")

	cmd := exec.Command("sh", "-n", file)
	require.NoError(t, cmd.Run())

	for _, arch := range architectures {
		t.Run(arch.alpine, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", `. "$1"; printf '%s\n%s\n%s\n' "$_source" "$_url" "$sha512sums"`, "sh", file)
			cmd.Env = append(os.Environ(), "CARCH="+arch.alpine)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			require.Len(t, lines, 3)
			require.Equal(t, "foo-1.2.3-"+arch.alpine+".tar.gz", lines[0])
			require.Contains(t, lines[1], "https://example.com/releases/foo_")
			require.Len(t, strings.Fields(lines[2])[0], 128)
			require.Equal(t, lines[0], strings.Fields(lines[2])[1])
		})
	}

	generated := ctx.Artifacts.Filter(artifact.ByType(artifact.APKBuild)).List()
	require.Len(t, generated, 1)
	require.Equal(t, "APKBUILD", generated[0].Name)
}

func TestRunAndPublish(t *testing.T) {
	repoURL := testlib.GitMakeBareRepository(t)
	key := testlib.MakeNewSSHKey(t, "")
	dist := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        dist,
			ProjectName: "foo",
			APKBuilds: []config.APKBuild{{
				Description: "Foo command",
				Homepage:    "https://example.com",
				License:     "MIT",
				URLTemplate: "https://example.com/{{ .ArtifactName }}",
				GitURL:      repoURL,
				PrivateKey:  key,
				Directory:   "testing/foo",
			}},
		},
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithSemver(1, 0, 0, ""),
		testctx.WithVersion("1.0.0"),
	)
	addArtifact(t, ctx, artifact.Artifact{
		Name:    "foo_linux_amd64",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableBinary,
		Extra: map[string]any{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "binary",
			artifact.ExtraBinary: "foo",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client.NewMock()))
	require.NoError(t, Pipe{}.Publish(ctx))

	cloneDir := t.TempDir()
	_, err := git.Run(t.Context(), "-C", cloneDir, "clone", repoURL, "repo")
	require.NoError(t, err)
	published := filepath.Join(cloneDir, "repo", "testing", "foo", "APKBUILD")
	require.FileExists(t, published)
	content, err := os.ReadFile(published)
	require.NoError(t, err)
	require.Contains(t, string(content), `install -Dm755 "$srcdir/$_source" "$pkgdir/usr/bin/foo"`)
}

func TestRunErrors(t *testing.T) {
	t.Run("no artifacts", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "foo",
			APKBuilds:   []config.APKBuild{{}},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.ErrorIs(t, runAll(ctx, client.NewMock()), ErrNoArchivesFound)
	})

	t.Run("duplicate architecture", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:        dist,
			ProjectName: "foo",
			APKBuilds: []config.APKBuild{{
				URLTemplate: "https://example.com/{{ .ArtifactName }}",
			}},
		}, testctx.WithVersion("1.0.0"))
		for i := range 2 {
			addArtifact(t, ctx, artifact.Artifact{
				Name:    fmt.Sprintf("foo-%d.tar.gz", i),
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.ErrorContains(t, runAll(ctx, client.NewMock()), "multiple artifacts found for Alpine architecture x86_64")
	})

	t.Run("invalid name template", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			Dist:        dist,
			ProjectName: "foo",
			APKBuilds: []config.APKBuild{{
				Name:        "{{ .Invalid }",
				URLTemplate: "https://example.com/{{ .ArtifactName }}",
			}},
		}, testctx.WithVersion("1.0.0"))
		addArtifact(t, ctx, artifact.Artifact{
			Name:    "foo.tar.gz",
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraFormat:   "tar.gz",
				artifact.ExtraBinaries: []string{"foo"},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, runAll(ctx, client.NewMock()))
	})
}

func TestPublishSkip(t *testing.T) {
	for name, setup := range map[string]func(*context.Context){
		"true": func(ctx *context.Context) {
			ctx.Config.APKBuilds[0].SkipUpload = "true"
		},
		"auto prerelease": func(ctx *context.Context) {
			ctx.Config.APKBuilds[0].SkipUpload = "auto"
			ctx.Semver.Prerelease = "beta.1"
		},
		"template": func(ctx *context.Context) {
			ctx.Config.APKBuilds[0].SkipUpload = "{{ .IsSnapshot }}"
			ctx.Snapshot = true
		},
	} {
		t.Run(name, func(t *testing.T) {
			dist := t.TempDir()
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist:        dist,
				ProjectName: "foo",
				APKBuilds: []config.APKBuild{{
					URLTemplate: "https://example.com/{{ .ArtifactName }}",
				}},
			}, testctx.WithVersion("1.0.0"), testctx.WithSemver(1, 0, 0, ""))
			setup(ctx)
			addArtifact(t, ctx, artifact.Artifact{
				Name:    "foo",
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableBinary,
				Extra: map[string]any{
					artifact.ExtraFormat: "binary",
					artifact.ExtraBinary: "foo",
				},
			})
			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, runAll(ctx, client.NewMock()))
			testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
		})
	}
}

func TestPartialSkip(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		APKBuilds: []config.APKBuild{
			{Disable: "true"},
			{Disable: "true"},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.True(t, pipe.IsSkip(runAll(ctx, client.NewMock())))
}

func addArtifact(tb testing.TB, ctx *context.Context, art artifact.Artifact) {
	tb.Helper()
	art.Path = filepath.Join(ctx.Config.Dist, "artifacts", art.Name)
	require.NoError(tb, os.MkdirAll(filepath.Dir(art.Path), 0o755))
	require.NoError(tb, os.WriteFile(art.Path, []byte(art.Name), 0o644))
	ctx.Artifacts.Add(&art)
}
