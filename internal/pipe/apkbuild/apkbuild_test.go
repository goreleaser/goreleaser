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

func TestToAPKVersion(t *testing.T) {
	for input, expected := range map[string]string{
		"1.2.3":           "1.2.3",
		"1.2.3-beta.1":    "1.2.3_beta1",
		"1.2.3-alpha1":    "1.2.3_alpha1",
		"1.2.3-beta1":     "1.2.3_beta1",
		"1.2.3-rc1":       "1.2.3_rc1",
		"1.2.3-rc.2":      "1.2.3_rc2",
		"1.2.3-preview.1": "1.2.3_pre_p0_p1",
		"1.2.3-rc-1":      "1.2.3_rc_p1",
	} {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, toAPKVersion(input))
		})
	}

	valid := `^[0-9]+(?:\.[0-9]+)*(?:_(?:alpha|beta|pre|rc|cvs|svn|git|hg|p)[0-9]*)*$`
	for _, version := range []string{
		"1.2.3+build.1",
		"1.2.3-SNAPSHOT-deadbeef",
	} {
		t.Run(version+" is valid", func(t *testing.T) {
			require.Regexp(t, valid, toAPKVersion(version))
		})
	}

	require.NotEqual(t, toAPKVersion("1.2.3-beta.1.0"), toAPKVersion("1.2.3-beta.10"))
	require.LessOrEqual(t, len(toAPKVersionNumber("SNAPSHOT-deadbeef")), 20)
}

func TestToAPKVersionOrder(t *testing.T) {
	testlib.CheckPath(t, "apk")
	versions := []string{
		"1.2.3-alpha1",
		"1.2.3-beta1",
		"1.2.3-rc1",
		"1.2.3",
	}
	for i := 1; i < len(versions); i++ {
		older := toAPKVersion(versions[i-1])
		newer := toAPKVersion(versions[i])
		output, err := exec.CommandContext(t.Context(), "apk", "version", "--test", older, newer).CombinedOutput()
		require.NoError(t, err, string(output))
		require.Equalf(t, "<", strings.TrimSpace(string(output)), "%s should sort before %s", older, newer)
	}
}

func TestDefaultPackage(t *testing.T) {
	for name, tt := range map[string]struct {
		artifact artifact.Artifact
		expect   string
	}{
		"binary path": {
			artifact: artifact.Artifact{
				Type: artifact.UploadableBinary,
				Extra: map[string]any{
					artifact.ExtraBinary: "bin/foo",
				},
			},
			expect: `install -Dm755 "$srcdir/$_source" "$pkgdir/usr/bin/foo"`,
		},
		"archive path": {
			artifact: artifact.Artifact{
				Type: artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraWrappedIn: "wrapped",
					artifact.ExtraBinaries:  []string{"bin/foo"},
				},
			},
			expect: `install -Dm755 "$srcdir/wrapped/bin/foo" "$pkgdir/usr/bin/foo"`,
		},
		"stripped archive path": {
			artifact: artifact.Artifact{
				Type: artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraWrappedIn: "wrapped",
					artifact.ExtraBinaries:  []string{"foo"},
				},
			},
			expect: `install -Dm755 "$srcdir/wrapped/foo" "$pkgdir/usr/bin/foo"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.expect, defaultPackage(&tt.artifact))
		})
	}
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		APKBuilds: []config.APKBuild{
			{},
			{Options: []string{"!strip"}},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))

	got := ctx.Config.APKBuilds[0]
	require.Equal(t, "foo", got.Name)
	require.Equal(t, "0", got.Rel)
	require.Equal(t, []string{"!check"}, got.Options)
	require.Equal(t, "v1", got.Goamd64)
	require.Equal(t, defaultCommitMsg, got.CommitMessageTemplate)
	require.Equal(t, "goreleaserbot", got.CommitAuthor.Name)
	require.Equal(t, "bot@goreleaser.com", got.CommitAuthor.Email)
	require.Equal(t, []string{"!strip", "!check"}, ctx.Config.APKBuilds[1].Options)
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
				artifact.ExtraID:        "foo",
				artifact.ExtraFormat:    "tar.gz",
				artifact.ExtraWrappedIn: "wrapped-" + arch.alpine,
				artifact.ExtraBinaries:  []string{"foo"},
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
	require.Contains(t, string(content), `install -Dm755 "$srcdir/wrapped-x86_64/foo" "$pkgdir/usr/bin/foo"`)
	require.Contains(t, string(content), "\t\tx86_64)\n\t\t\tinstall -Dm755")
	require.NotContains(t, string(content), "ignored_v3")

	testlib.CheckPath(t, "sh")
	cmd := exec.CommandContext(t.Context(), "sh", "-n", file)
	require.NoError(t, cmd.Run())

	for _, arch := range architectures {
		t.Run(arch.alpine, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "sh", "-c", `. "$1"; printf '%s\n%s\n%s\n' "$_source" "$_url" "$sha512sums"`, "sh", file)
			cmd.Env = append(os.Environ(), "CARCH="+arch.alpine)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			require.Len(t, lines, 3)
			require.Equal(t, "foo-1.2.3-"+arch.alpine+".tar.gz", lines[0])
			require.Contains(t, lines[1], "https://example.com/releases/foo_")
			require.Len(t, strings.Fields(lines[2])[0], 128)
			require.Equal(t, lines[0], strings.Fields(lines[2])[1])

			cmd = exec.CommandContext(t.Context(), "sh", "-c", `. "$1"; install() { printf '%s\n' "$@"; }; package`, "sh", file)
			cmd.Env = append(os.Environ(), "CARCH="+arch.alpine, "srcdir=/src", "pkgdir=/pkg")
			out, err = cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			require.Equal(t, []string{
				"-Dm755",
				"/src/wrapped-" + arch.alpine + "/foo",
				"/pkg/usr/bin/foo",
			}, strings.Split(strings.TrimSpace(string(out)), "\n"))
		})
	}

	generated := ctx.Artifacts.Filter(artifact.ByType(artifact.APKBuild)).List()
	require.Len(t, generated, 1)
	require.Equal(t, "APKBUILD", generated[0].Name)
}

func TestRunAndPublish(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)
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
	summary, err := os.ReadFile(summaryFile)
	require.NoError(t, err)
	require.Contains(t, string(summary), "Pushed Alpine Linux package `foo` to `"+repoURL+"`")
}

func TestDuplicateNamesPublishIndependentlyToDefaultBranches(t *testing.T) {
	masterRepo := testlib.GitMakeBareRepository(t)
	mainRepo := testlib.GitMakeBareRepository(t)
	_, err := git.Run(t.Context(), "-C", mainRepo, "symbolic-ref", "HEAD", "refs/heads/main")
	require.NoError(t, err)

	key := testlib.MakeNewSSHKey(t, "")
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        t.TempDir(),
			ProjectName: "foo",
			APKBuilds: []config.APKBuild{
				{
					Description: "Master package",
					Homepage:    "https://example.com/master",
					License:     "MIT",
					URLTemplate: "https://example.com/{{ .ArtifactName }}",
					GitURL:      masterRepo,
					PrivateKey:  key,
					Directory:   "testing/foo",
				},
				{
					Description: "Main package",
					Homepage:    "https://example.com/main",
					License:     "Apache-2.0",
					URLTemplate: "https://example.com/{{ .ArtifactName }}",
					GitURL:      mainRepo,
					PrivateKey:  key,
					Directory:   "community/foo",
				},
			},
		},
		testctx.WithCurrentTag("v1.0.0"),
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
	generated := ctx.Artifacts.Filter(artifact.ByType(artifact.APKBuild)).List()
	require.Len(t, generated, 2)
	require.NotEqual(t, generated[0].Path, generated[1].Path)
	require.NoError(t, Pipe{}.Publish(ctx))

	masterContent := testlib.CatFileFromBareRepositoryOnBranch(t, masterRepo, "master", "testing/foo/APKBUILD")
	require.Contains(t, string(masterContent), `pkgdesc='Master package'`)
	mainContent := testlib.CatFileFromBareRepositoryOnBranch(t, mainRepo, "main", "community/foo/APKBUILD")
	require.Contains(t, string(mainContent), `pkgdesc='Main package'`)
}

func TestArchiveFormats(t *testing.T) {
	for _, tt := range []struct {
		format  string
		wantErr bool
	}{
		{format: "tar"},
		{format: "tgz"},
		{format: "tar.gz"},
		{format: "tar.xz"},
		{format: "zip"},
		{format: "gz", wantErr: true},
		{format: "xz", wantErr: true},
		{format: "txz", wantErr: true},
		{format: "tzst", wantErr: true},
	} {
		t.Run(tt.format, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist:        t.TempDir(),
				ProjectName: "foo",
				APKBuilds: []config.APKBuild{{
					Description: "Foo",
					Homepage:    "https://example.com",
					License:     "MIT",
					URLTemplate: "https://example.com/{{ .ArtifactName }}",
				}},
			}, testctx.WithVersion("1.0.0"))
			addArtifact(t, ctx, artifact.Artifact{
				Name:    "foo." + tt.format,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraFormat:   tt.format,
					artifact.ExtraBinaries: []string{"foo"},
				},
			})
			require.NoError(t, Pipe{}.Default(ctx))
			err := runAll(ctx, client.NewMock())
			if tt.wantErr {
				require.ErrorIs(t, err, ErrNoArchivesFound)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMultipleArchiveFormatsUseDeterministicChoice(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		APKBuilds: []config.APKBuild{{
			IDs:         []string{"foo"},
			Description: "Foo",
			Homepage:    "https://example.com",
			License:     "MIT",
			URLTemplate: "https://example.com/{{ .ArtifactName }}",
		}},
	}, testctx.WithVersion("1.0.0"))
	for _, format := range []string{"zip", "tar.gz"} {
		addArtifact(t, ctx, artifact.Artifact{
			Name:    "foo." + format,
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:       "foo",
				artifact.ExtraFormat:   format,
				artifact.ExtraBinaries: []string{"foo"},
			},
		})
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client.NewMock()))
	content, err := os.ReadFile(filepath.Join(ctx.Config.Dist, "apkbuild", "foo.apkbuild"))
	require.NoError(t, err)
	require.Contains(t, string(content), "foo.tar.gz")
	require.NotContains(t, string(content), "foo.zip")
}

func TestArchivePreferredOverBinaryForSameID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		APKBuilds: []config.APKBuild{{
			IDs:         []string{"foo"},
			Description: "Foo",
			Homepage:    "https://example.com",
			License:     "MIT",
			URLTemplate: "https://example.com/{{ .ArtifactName }}",
		}},
	}, testctx.WithVersion("1.0.0"))
	for _, art := range []artifact.Artifact{
		{
			Name: "foo", Goos: "linux", Goarch: "amd64", Goamd64: "v1", Type: artifact.UploadableBinary,
			Extra: map[string]any{artifact.ExtraID: "foo", artifact.ExtraFormat: "binary", artifact.ExtraBinary: "foo"},
		},
		{
			Name: "foo.tar.gz", Goos: "linux", Goarch: "amd64", Goamd64: "v1", Type: artifact.UploadableArchive,
			Extra: map[string]any{artifact.ExtraID: "foo", artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
		},
	} {
		addArtifact(t, ctx, art)
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client.NewMock()))
	content, err := os.ReadFile(filepath.Join(ctx.Config.Dist, "apkbuild", "foo.apkbuild"))
	require.NoError(t, err)
	require.Contains(t, string(content), "foo.tar.gz")
}

func TestConfiguredGoamd64(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		APKBuilds: []config.APKBuild{{
			Goamd64:     "v3",
			Description: "Foo",
			Homepage:    "https://example.com",
			License:     "MIT",
			URLTemplate: "https://example.com/{{ .ArtifactName }}",
		}},
	}, testctx.WithVersion("1.0.0"))
	addArtifact(t, ctx, artifact.Artifact{
		Name: "foo-v3.tar.gz", Goos: "linux", Goarch: "amd64", Goamd64: "v3", Type: artifact.UploadableArchive,
		Extra: map[string]any{artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client.NewMock()))
}

func TestWhitespacePackageUsesInferredInstructions(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		APKBuilds: []config.APKBuild{{
			Description: "Foo",
			Homepage:    "https://example.com",
			License:     "MIT",
			URLTemplate: "https://example.com/{{ .ArtifactName }}",
			Package:     "{{ if .IsSnapshot }}custom{{ else }} \n\t {{ end }}",
		}},
	}, testctx.WithVersion("1.0.0"))
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
	content, err := os.ReadFile(filepath.Join(ctx.Config.Dist, "apkbuild", "foo.apkbuild"))
	require.NoError(t, err)
	require.Contains(t, string(content), `install -Dm755 "$srcdir/$_source" "$pkgdir/usr/bin/foo"`)
}

func TestRequiredMetadata(t *testing.T) {
	for name, tt := range map[string]struct {
		setup   func(*config.APKBuild)
		wantErr string
	}{
		"description": {
			setup:   func(cfg *config.APKBuild) { cfg.Description = "" },
			wantErr: "apkbuild.description is required",
		},
		"homepage": {
			setup:   func(cfg *config.APKBuild) { cfg.Homepage = "" },
			wantErr: "apkbuild.homepage is required",
		},
		"license": {
			setup:   func(cfg *config.APKBuild) { cfg.License = "" },
			wantErr: "apkbuild.license is required",
		},
		"maintainers": {
			setup: func(cfg *config.APKBuild) {
				cfg.Maintainers = []string{"One <one@example.com>", "Two <two@example.com>"}
			},
			wantErr: "apkbuild.maintainers must contain at most one entry",
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := config.APKBuild{
				Description: "Foo",
				Homepage:    "https://example.com",
				License:     "MIT",
				URLTemplate: "https://example.com/{{ .ArtifactName }}",
			}
			tt.setup(&cfg)
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist:        t.TempDir(),
				ProjectName: "foo",
				APKBuilds:   []config.APKBuild{cfg},
			}, testctx.WithVersion("1.0.0"))
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
			require.EqualError(t, runAll(ctx, client.NewMock()), tt.wantErr)
		})
	}
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
				Description: "Foo",
				Homepage:    "https://example.com",
				License:     "MIT",
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
					Description: "Foo",
					Homepage:    "https://example.com",
					License:     "MIT",
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
