package aur

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

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
	pkg, err := applyTemplate(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), aurTemplateData, data)
	require.NoError(t, err)

	golden.RequireEqual(t, []byte(pkg))
}

func TestAurSimple(t *testing.T) {
	pkg, err := applyTemplate(testctx.New(), aurTemplateData, createTemplateData())
	require.NoError(t, err)
	require.Contains(t, pkg, `# Maintainer: Ciclano <ciclano@example.com>`)
	require.Contains(t, pkg, `# Maintainer: Cicrano <cicrano@example.com>`)
	require.Contains(t, pkg, `# Contributor: Fulano <fulano@example.com>`)
	require.Contains(t, pkg, `# Contributor: Beltrano <beltrano@example.com>`)
	require.Contains(t, pkg, `pkgname='test-bin'`)
	require.Contains(t, pkg, `url='https://example.com'`)
	require.Contains(t, pkg, `source_x86_64=("${pkgname}_${pkgver}_x86_64.tar.gz::https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz")`)
	require.Contains(t, pkg, `sha256sums_x86_64=('1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67')`)
	require.Contains(t, pkg, `pkgver=0.1.3`)
}

func TestFullSrcInfo(t *testing.T) {
	data := createTemplateData()
	data.License = "MIT"
	pkg, err := applyTemplate(testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	}), srcInfoTemplate, data)
	require.NoError(t, err)

	golden.RequireEqual(t, []byte(pkg))
}

func TestSrcInfoSimple(t *testing.T) {
	pkg, err := applyTemplate(testctx.New(), srcInfoTemplate, createTemplateData())
	require.NoError(t, err)
	require.Contains(t, pkg, `pkgbase = test-bin`)
	require.Contains(t, pkg, `pkgname = test-bin`)
	require.Contains(t, pkg, `url = https://example.com`)
	require.Contains(t, pkg, `source_x86_64 = https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz`)
	require.Contains(t, pkg, `sha256sums_x86_64 = 1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67`)
	require.Contains(t, pkg, `pkgver = 0.1.3`)
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare                func(ctx *context.Context)
		expectedRunError       string
		expectedPublishError   string
		expectedPublishErrorIs error
		expectedErrorCheck     func(testing.TB, error)
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.AURs[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"with-more-opts": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.AURs[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.AURs[0].Maintainers = []string{"me"}
				ctx.Config.AURs[0].Contributors = []string{"me as well"}
				ctx.Config.AURs[0].Depends = []string{"curl", "bash"}
				ctx.Config.AURs[0].OptDepends = []string{"wget: stuff", "foo: bar"}
				ctx.Config.AURs[0].Provides = []string{"git", "svn"}
				ctx.Config.AURs[0].Conflicts = []string{"libcurl", "cvs", "blah"}
			},
		},
		"default-gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.AURs[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid-name-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].Name = "{{ .Asdsa }"
			},
			expectedRunError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid-package-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].Package = "{{ .Asdsa }"
			},
			expectedRunError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid-commit-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-key-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].PrivateKey = "{{ .Asdsa }"
			},
			expectedErrorCheck: testlib.RequireTemplateError,
		},
		"no-key": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].PrivateKey = ""
			},
			expectedPublishError: `private_key is empty`,
		},
		"key-not-found": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].PrivateKey = "testdata/nope"
			},
			expectedPublishErrorIs: os.ErrNotExist,
		},
		"invalid-git-url-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].GitURL = "{{ .Asdsa }"
			},
			expectedErrorCheck: testlib.RequireTemplateError,
		},
		"no-git-url": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].GitURL = ""
			},
			expectedPublishError: `url is empty`,
		},
		"invalid-ssh-cmd-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].GitSSHCommand = "{{ .Asdsa }"
			},
			expectedErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-commit-author-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURs[0].CommitAuthor.Name = "{{ .Asdsa }"
			},
			expectedErrorCheck: testlib.RequireTemplateError,
		},
	} {
		t.Run(name, func(t *testing.T) {
			url := testlib.GitMakeBareRpository(t)
			key := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")

			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: name,
					AURs: []config.AUR{
						{
							Name:        name,
							IDs:         []string{"foo"},
							PrivateKey:  key,
							License:     "MIT",
							GitURL:      url,
							Description: "A run pipe test fish food and FOO={{ .Env.FOO }}",
						},
					},
					Env: []string{"FOO=foo_is_bar"},
				},
				testctx.WithCurrentTag("v1.0.1-foo"),
				testctx.WithSemver(1, 0, 1, "foo"),
				testctx.WithVersion("1.0.1-foo"),
			)

			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "should-be-ignored.tar.gz",
				Path:    "doesnt matter",
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v3",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:       "bar",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"bar"},
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bar_bin.tar.gz",
				Path:    "doesnt matter",
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:       "bar",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"bar"},
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "bin.tar.gz",
				Path:    path,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Type:    artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:       "foo",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"name"},
				},
			})

			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			client := client.NewMock()

			require.NoError(t, Pipe{}.Default(ctx))

			if tt.expectedRunError != "" {
				require.EqualError(t, runAll(ctx, client), tt.expectedRunError)
				return
			}
			require.NoError(t, runAll(ctx, client))

			if tt.expectedPublishError != "" {
				require.EqualError(t, Pipe{}.Publish(ctx), tt.expectedPublishError)
				return
			}

			if tt.expectedPublishErrorIs != nil {
				require.ErrorIs(t, Pipe{}.Publish(ctx), tt.expectedPublishErrorIs)
				return
			}

			if tt.expectedErrorCheck != nil {
				tt.expectedErrorCheck(t, Pipe{}.Publish(ctx))
				return
			}

			require.NoError(t, Pipe{}.Publish(ctx))

			requireEqualRepoFiles(t, folder, name, url)
		})
	}
}

func TestRunPipe(t *testing.T) {
	url := testlib.GitMakeBareRpository(t)
	key := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")

	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURs: []config.AUR{
				{
					License:     "MIT",
					Description: "A run pipe test aur and FOO={{ .Env.FOO }}",
					Homepage:    "https://github.com/goreleaser",
					IDs:         []string{"foo"},
					GitURL:      url,
					PrivateKey:  key,
				},
			},
			GitHubURLs: config.GitHubURLs{
				Download: "https://github.com",
			},
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "test",
					Name:  "test",
				},
			},
			Env: []string{"FOO=foo_is_bar"},
		},
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.0.1"),
		testctx.WithSemver(1, 0, 1, ""),
		testctx.WithVersion("1.0.1"),
	)

	for _, a := range []struct {
		name   string
		goos   string
		goarch string
		goarm  string
	}{
		{
			name:   "bin",
			goos:   "darwin",
			goarch: "amd64",
		},
		{
			name:   "bin",
			goos:   "darwin",
			goarch: "arm64",
		},
		{
			name:   "bin",
			goos:   "windows",
			goarch: "arm64",
		},
		{
			name:   "bin",
			goos:   "windows",
			goarch: "amd64",
		},
		{
			name:   "bin",
			goos:   "linux",
			goarch: "386",
		},
		{
			name:   "bin",
			goos:   "linux",
			goarch: "amd64",
		},
		{
			name:   "arm64",
			goos:   "linux",
			goarch: "arm64",
		},
		{
			name:   "armv5",
			goos:   "linux",
			goarch: "arm",
			goarm:  "5",
		},
		{
			name:   "armv6",
			goos:   "linux",
			goarch: "arm",
			goarm:  "6",
		},
		{
			name:   "armv7",
			goos:   "linux",
			goarch: "arm",
			goarm:  "7",
		},
	} {
		path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:    fmt.Sprintf("%s.tar.gz", a.name),
			Path:    path,
			Goos:    a.goos,
			Goarch:  a.goarch,
			Goarm:   a.goarm,
			Goamd64: "v1",
			Type:    artifact.UploadableArchive,
			Extra: map[string]interface{}{
				artifact.ExtraID:       "foo",
				artifact.ExtraFormat:   "tar.gz",
				artifact.ExtraBinaries: []string{"foo"},
			},
		})
		f, err := os.Create(path)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	client := client.NewMock()

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, Pipe{}.Publish(ctx))

	requireEqualRepoFiles(t, folder, "foo", url)
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		AURs:        []config.AUR{{}},
	}, testctx.GitHubTokenType)
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeBinaryRelease(t *testing.T) {
	url := testlib.GitMakeBareRpository(t)
	key := testlib.MakeNewSSHKey(t, keygen.Ed25519, "")
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURs: []config.AUR{{
				GitURL:     url,
				PrivateKey: key,
			}},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
		testctx.WithSemver(1, 2, 1, ""),
	)

	path := filepath.Join(folder, "dist/foo_linux_amd64/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_linux_amd64",
		Path:    path,
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID:     "foo",
			artifact.ExtraFormat: "binary",
			artifact.ExtraBinary: "foo",
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, Pipe{}.Publish(ctx))

	requireEqualRepoFiles(t, folder, "foo", url)
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
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
		Extra: map[string]interface{}{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	client := client.NewMock()

	assertNoPublish := func(t *testing.T) {
		t.Helper()
		require.NoError(t, runAll(ctx, client))
		testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
		require.False(t, client.CreatedFile)
	}
	t.Run("skip upload true", func(t *testing.T) {
		ctx.Config.AURs[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.AURs[0].SkipUpload = "auto"
		ctx.Semver.Prerelease = "beta1"
		assertNoPublish(t)
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
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
		Extra: map[string]interface{}{
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
		ctx := testctx.NewWithCfg(config.Project{
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
		ctx := testctx.NewWithCfg(config.Project{
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
		ctx := testctx.NewWithCfg(config.Project{
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
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			AURs: []config.AUR{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func requireEqualRepoFiles(tb testing.TB, folder, name, url string) {
	tb.Helper()
	dir := tb.TempDir()
	_, err := git.Run(testctx.New(), "-C", dir, "clone", url, "repo")
	require.NoError(tb, err)

	for reponame, ext := range map[string]string{
		"PKGBUILD": ".pkgbuild",
		".SRCINFO": ".srcinfo",
	} {
		path := filepath.Join(folder, "aur", name+"-bin"+ext)
		bts, err := os.ReadFile(path)
		require.NoError(tb, err)
		golden.RequireEqualExt(tb, bts, ext)

		bts, err = os.ReadFile(filepath.Join(dir, "repo", reponame))
		require.NoError(tb, err)
		golden.RequireEqualExt(tb, bts, ext)
	}
}
