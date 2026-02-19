package aur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
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

func createTemplateData() templateData {
	return templateData{
		Name:       "test-bin",
		Desc:       "Some desc",
		Homepage:   "https://example.com",
		Conflicts:  []string{"nope"},
		Depends:    []string{"nope"},
		Arches:     []string{"x86_64", "i686", "aarch64", "armv6h", "armv7h"},
		Rel:        "1",
		Provides:   []string{"test"},
		OptDepends: []string{"nfpm"},
		Backup: []string{
			"/etc/mypkg.conf",
			"/var/share/mypkg",
		},
		Maintainers: []string{
			"Ciclano <ciclano@example.com>",
			"Cicrano <cicrano@example.com>",
		},
		Contributors: []string{
			"Fulano <fulano@example.com>",
			"Beltrano <beltrano@example.com>",
		},
		License: "MIT",
		Version: "0.1.3",
		Install: "./testdata/install.sh",
		Package: `# bin
		install -Dm755 "./goreleaser" "${pkgdir}/usr/bin/goreleaser"

		# license
		install -Dm644 "./LICENSE.md" "${pkgdir}/usr/share/licenses/goreleaser/LICENSE"

		# completions
		mkdir -p "${pkgdir}/usr/share/bash-completion/completions/"
		mkdir -p "${pkgdir}/usr/share/zsh/site-functions/"
		mkdir -p "${pkgdir}/usr/share/fish/vendor_completions.d/"
		install -Dm644 "./completions/goreleaser.bash" "${pkgdir}/usr/share/bash-completion/completions/goreleaser"
		install -Dm644 "./completions/goreleaser.zsh" "${pkgdir}/usr/share/zsh/site-functions/_goreleaser"
		install -Dm644 "./completions/goreleaser.fish" "${pkgdir}/usr/share/fish/vendor_completions.d/goreleaser.fish"

		# man pages
		install -Dm644 "./manpages/goreleaser.1.gz" "${pkgdir}/usr/share/man/man1/goreleaser.1.gz"`,
		ReleasePackages: []releasePackage{
			{
				Arch:        "x86_64",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
				Format:      "tar.gz",
			},
			{
				Arch:        "armv6h",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_Arm6.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
				Format:      "tar.gz",
			},
			{
				Arch:        "aarch64",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_Arm64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
				Format:      "tar.gz",
			},
			{
				Arch:        "i686",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_386.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
				Format:      "tar.gz",
			},
			{
				Arch:        "armv7h",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_arm7.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
				Format:      "tar.gz",
			},
		},
	}
}

func TestFullAur(t *testing.T) {
	data := createTemplateData()
	pkg, err := applyTemplate(testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
	}), aurTemplateData, data)
	require.NoError(t, err)

	golden.RequireEqual(t, []byte(pkg))
}

func TestAurSimple(t *testing.T) {
	pkg, err := applyTemplate(testctx.Wrap(t.Context()), aurTemplateData, createTemplateData())
	require.NoError(t, err)
	require.Contains(t, pkg, `# Maintainer: Ciclano <ciclano@example.com>`)
	require.Contains(t, pkg, `# Maintainer: Cicrano <cicrano@example.com>`)
	require.Contains(t, pkg, `# Contributor: Fulano <fulano@example.com>`)
	require.Contains(t, pkg, `# Contributor: Beltrano <beltrano@example.com>`)
	require.Contains(t, pkg, `pkgname='test-bin'`)
	require.Contains(t, pkg, `url='https://example.com'`)
	require.Contains(t, pkg, `source_x86_64=("${pkgname}_${pkgver}_x86_64.tar.gz::https://github.com/caarlos0/test/releases/download/v${pkgver}/test_Linux_x86_64.tar.gz")`)
	require.Contains(t, pkg, `sha256sums_x86_64=('1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67')`)
	require.Contains(t, pkg, `pkgver=0.1.3`)
}

func TestFullSrcInfo(t *testing.T) {
	data := createTemplateData()
	data.License = "MIT"
	pkg, err := applyTemplate(testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
	}), srcInfoTemplate, data)
	require.NoError(t, err)

	golden.RequireEqual(t, []byte(pkg))
}

func TestSrcInfoSimple(t *testing.T) {
	pkg, err := applyTemplate(testctx.Wrap(t.Context()), srcInfoTemplate, createTemplateData())
	require.NoError(t, err)
	require.Contains(t, pkg, `pkgbase = test-bin`)
	require.Contains(t, pkg, `pkgname = test-bin`)
	require.Contains(t, pkg, `url = https://example.com`)
	require.Contains(t, pkg, `source_x86_64 = https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz`)
	require.Contains(t, pkg, `sha256sums_x86_64 = 1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67`)
	require.Contains(t, pkg, `pkgver = 0.1.3`)
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		AURs:        []config.AUR{{}},
	}, testctx.GitHubTokenType)
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	testPublish := func(tb testing.TB, modifier func(ctx *context.Context)) {
		tb.Helper()
		ctx := testctx.WrapWithCfg(t.Context(),
			config.Project{
				Dist:        folder,
				ProjectName: "foo",
				Release:     config.Release{},
				AURs:        []config.AUR{{}},
			},
			testctx.GitHubTokenType,
			testctx.WithCurrentTag("v1.0.1"),
			testctx.WithSemver(1, 0, 1, ""),
		)

		path := filepath.Join(folder, "whatever.tar.gz")
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:    "bin",
			Path:    path,
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v1",
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:       "foo",
				artifact.ExtraFormat:   "tar.gz",
				artifact.ExtraBinaries: []string{"foo"},
			},
		})

		modifier(ctx)

		require.NoError(t, Pipe{}.Default(ctx))
		client := client.NewMock()
		require.NoError(t, runAll(ctx, client))
		t.Log(Pipe{}.Publish(ctx))
		testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
		require.False(t, client.CreatedFile)
	}

	t.Run("skip upload true", func(t *testing.T) {
		testPublish(t, func(ctx *context.Context) {
			ctx.Config.AURs[0].SkipUpload = "true"
			ctx.Semver.Prerelease = ""
		})
	})
	t.Run("skip upload auto", func(t *testing.T) {
		testPublish(t, func(ctx *context.Context) {
			ctx.Config.AURs[0].SkipUpload = "auto"
			ctx.Semver.Prerelease = "beta1"
		})
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Release:     config.Release{},
			AURs: []config.AUR{
				{},
			},
		},
		testctx.WithGitInfo(context.GitInfo{CurrentTag: "v1.0.1"}),
		testctx.WithSemver(1, 0, 1, ""),
	)
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})
	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "myproject",
			AURs:        []config.AUR{{}},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AUR{
			Name:                  "myproject-bin",
			Conflicts:             []string{"myproject"},
			Provides:              []string{"myproject"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v1",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURs[0])
	})

	t.Run("name-without-bin-suffix", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "myproject",
			AURs: []config.AUR{
				{
					Name: "foo",
				},
			},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AUR{
			Name:                  "foo-bin",
			Conflicts:             []string{"myproject"},
			Provides:              []string{"myproject"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v1",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURs[0])
	})

	t.Run("partial", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "myproject",
			AURs: []config.AUR{
				{
					Conflicts: []string{"somethingelse"},
					Goamd64:   "v3",
				},
			},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AUR{
			Name:                  "myproject-bin",
			Conflicts:             []string{"somethingelse"},
			Provides:              []string{"myproject"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v3",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURs[0])
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			AURs: []config.AUR{
				{},
			},
		}, testctx.Skip(skips.AUR))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			AURs: []config.AUR{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
