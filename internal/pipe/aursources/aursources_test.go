package aursources

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
		Name:        "test",
		Desc:        "Some desc",
		Homepage:    "https://example.com",
		Conflicts:   []string{"nope"},
		Depends:     []string{"nope"},
		Arches:      []string{"x86_64", "i686", "aarch64", "armv6h", "armv7h"},
		Rel:         "1",
		Provides:    []string{"test"},
		OptDepends:  []string{"nfpm"},
		MakeDepends: []string{"git"},
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
		Prepare: `cd "${pkgname}_${pkgver}"
		# download dependencies
		go mod download`,
		Build: `  cd "${pkgname}_${pkgver}"
	  	export CGO_CPPFLAGS="${CPPFLAGS}"
	  	export CGO_CFLAGS="${CFLAGS}"
	  	export CGO_CXXFLAGS="${CXXFLAGS}"
	  	export CGO_LDFLAGS="${LDFLAGS}"
	  	export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
	  	go build -ldflags="-w -s -buildid='' -linkmode=external -X main.version=${pkgver}" .
	  	chmod +x "./${pkgname}"`,
		Package: `cd "${pkgname}_${pkgver}"
		install -Dsm755 ./goreleaser "${pkgdir}/usr/bin/goreleaser"
		mkdir -p "${pkgdir}/usr/share/bash-completion/completions/"
		mkdir -p "${pkgdir}/usr/share/zsh/site-functions/"
		mkdir -p "${pkgdir}/usr/share/fish/vendor_completions.d/"
		./goreleaser completion bash > "${pkgdir}/usr/share/bash-completion/completions/goreleaser"
		./goreleaser completion zsh > "${pkgdir}/usr/share/zsh/site-functions/_goreleaser"
		./goreleaser completion fish > "${pkgdir}/usr/share/fish/vendor_completions.d/goreleaser.fish"`,
		Sources: sources{
			DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
			SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			Format:      "tar.gz",
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
	require.Contains(t, pkg, `pkgname='test'`)
	require.Contains(t, pkg, `url='https://example.com'`)
	require.Contains(t, pkg, `source=("${pkgname}_${pkgver}.tar.gz::https://github.com/caarlos0/test/releases/download/v${pkgver}/test_Linux_x86_64.tar.gz")`)
	require.Contains(t, pkg, `sha256sums=('1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67')`)
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
	require.Contains(t, pkg, `pkgbase = test`)
	require.Contains(t, pkg, `pkgname = test`)
	require.Contains(t, pkg, `url = https://example.com`)
	require.Contains(t, pkg, `source = https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz`)
	require.Contains(t, pkg, `sha256sums = 1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67`)
	require.Contains(t, pkg, `pkgver = 0.1.3`)
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "foo",
		AURSources:  []config.AURSource{{}},
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
				AURSources:  []config.AURSource{{}},
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
			Name:    "source",
			Path:    path,
			Goamd64: "v1",
			Type:    artifact.UploadableSourceArchive,
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
			ctx.Config.AURSources[0].SkipUpload = "true"
			ctx.Semver.Prerelease = ""
		})
	})
	t.Run("skip upload auto", func(t *testing.T) {
		testPublish(t, func(ctx *context.Context) {
			ctx.Config.AURSources[0].SkipUpload = "auto"
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
			AURSources: []config.AURSource{
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
		Name:   "source",
		Path:   path,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableSourceArchive,
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
			AURSources:  []config.AURSource{{}},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AURSource{
			Name:                  "myproject",
			Conflicts:             []string{"myproject"},
			Provides:              []string{"myproject"},
			Arches:                []string{"x86_64", "aarch64"},
			MakeDepends:           []string{"go", "git"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v1",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURSources[0])
	})

	t.Run("name-with-bin-suffix", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "myproject-bin",
			AURSources: []config.AURSource{
				{
					Name: "foo",
				},
			},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AURSource{
			Name:                  "foo",
			Conflicts:             []string{"myproject-bin"}, // TODO(ldez) not sure about that.
			Provides:              []string{"myproject-bin"}, // TODO(ldez) not sure about that.
			Arches:                []string{"x86_64", "aarch64"},
			MakeDepends:           []string{"go", "git"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v1",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURSources[0])
	})

	t.Run("partial", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			ProjectName: "myproject",
			AURSources: []config.AURSource{
				{
					Conflicts: []string{"somethingelse"},
					Goamd64:   "v3",
				},
			},
		}, testctx.GitHubTokenType)
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.AURSource{
			Name:                  "myproject",
			Conflicts:             []string{"somethingelse"},
			Provides:              []string{"myproject"},
			Arches:                []string{"x86_64", "aarch64"},
			MakeDepends:           []string{"go", "git"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			Goamd64:               "v3",
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "bot@goreleaser.com",
			},
		}, ctx.Config.AURSources[0])
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			AURSources: []config.AURSource{
				{},
			},
		}, testctx.Skip(skips.AURSource))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			AURSources: []config.AURSource{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
