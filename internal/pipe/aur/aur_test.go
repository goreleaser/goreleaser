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
		Homepage:   "https://google.com",
		Conflicts:  []string{"nope"},
		Depends:    []string{"nope"},
		Arches:     []string{"x86_64", "i686", "aarch64", "armv6h", "armv7h"},
		Rel:        "1",
		Provides:   []string{"test"},
		OptDepends: []string{"nfpm"},
		Maintainer: "Ciclano <ciclano@oobar.as>",
		Contributors: []string{
			"Fulano <fulano@oobar.as>",
			"Beltrano <Beltrano@oobar.as>",
		},
		License: "MIT",
		Version: "0.1.3",
		Package: `
		# bin
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
		install -Dm644 "./manpages/goreleaser.1.gz" "${pkgdir}/usr/share/man/man1/goreleaser.1.gz"
		`,
		ReleasePackages: []releasePackage{
			{
				Arch:        "x86_64",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "armv6h",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_Arm6.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "aarch64",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_Arm64.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "i686",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_386.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
			{
				Arch:        "armv7h",
				DownloadURL: "https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_arm7.tar.gz",
				SHA256:      "1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67",
			},
		},
	}
}

func assertDefaultTemplateData(t *testing.T, pkgbuild string) {
	t.Helper()
	require.Contains(t, pkgbuild, `pkgname='test-bin'`)
	require.Contains(t, pkgbuild, `url='https://google.com'`)
	require.Contains(t, pkgbuild, `source_x86_64=('https://github.com/caarlos0/test/releases/download/v0.1.3/test_Linux_x86_64.tar.gz')`)
	require.Contains(t, pkgbuild, `sha256sums_x86_64=('1633f61598ab0791e213135923624eb342196b3494909c91899bcd0560f84c67')`)
	require.Contains(t, pkgbuild, `pkgver=0.1.3`)
}

func TestFullPkgBuild(t *testing.T) {
	data := createTemplateData()
	data.License = "MIT"
	pkg, err := doBuildPkgBuild(context.New(config.Project{
		ProjectName: "foo",
	}), data)
	require.NoError(t, err)

	golden.RequireEqual(t, []byte(pkg))
}

func TestPkgBuildSimple(t *testing.T) {
	pkg, err := doBuildPkgBuild(context.New(config.Project{}), createTemplateData())
	require.NoError(t, err)
	assertDefaultTemplateData(t, pkg)
}

func TestFullPipe(t *testing.T) {
	type testcase struct {
		prepare              func(ctx *context.Context)
		expectedPublishError string
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.PkgBuilds[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"default_gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.PkgBuilds[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid_commit_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid_key_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].PrivateKey = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"no_key": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].PrivateKey = ""
			},
			expectedPublishError: `pkgbuild.private_key is empty`,
		},
		"key_not_found": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].PrivateKey = "testdata/nope"
			},
			expectedPublishError: `key "testdata/nope" does not exist`,
		},
		"invalid_git_url_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].GitURL = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"no_git_url": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].GitURL = ""
			},
			expectedPublishError: `pkgbuild.git_url is empty`,
		},
		"invalid_ssh_cmd_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].SSHCommand = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
		"invalid_commit_author_template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.PkgBuilds[0].CommitAuthor.Name = "{{ .Asdsa }"
			},
			expectedPublishError: `template: tmpl:1: unexpected "}" in operand`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			url := makeBareRepo(t)
			key := makeKey(t)

			folder := t.TempDir()
			ctx := &context.Context{
				Git: context.GitInfo{
					CurrentTag: "v1.0.1",
				},
				Version:   "1.0.1",
				Artifacts: artifact.New(),
				Env: map[string]string{
					"FOO": "foo_is_bar",
				},
				Config: config.Project{
					Dist:        folder,
					ProjectName: name,
					PkgBuilds: []config.PkgBuild{
						{
							Name: name,
							IDs: []string{
								"foo",
							},
							PrivateKey:  key,
							License:     "MIT",
							GitURL:      url,
							Description: "A run pipe test fish food and FOO={{ .Env.FOO }}",
						},
					},
				},
			}
			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bar_bin.tar.gz",
				Path:   "doesnt matter",
				Goos:   "linux",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
				Extra: map[string]interface{}{
					artifact.ExtraID:       "bar",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"bar"},
				},
			})
			path := filepath.Join(folder, "bin.tar.gz")
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "bin.tar.gz",
				Path:   path,
				Goos:   "linux",
				Goarch: "amd64",
				Type:   artifact.UploadableArchive,
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
			distFile := filepath.Join(folder, "aur/pkgbuilds", name)

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, runAll(ctx, client))
			if tt.expectedPublishError != "" {
				require.EqualError(t, publishAll(ctx, client), tt.expectedPublishError)
				return
			}
			require.NoError(t, publishAll(ctx, client))

			distBts, err := os.ReadFile(distFile)
			require.NoError(t, err)

			golden.RequireEqual(t, distBts)
			golden.RequireEqual(t, cloneAndGetPKGBUILD(t, url))
		})
	}
}

func TestRunPipe(t *testing.T) {
	url := makeBareRepo(t)
	key := makeKey(t)

	folder := t.TempDir()
	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Git: context.GitInfo{
			CurrentTag: "v1.0.1",
		},
		Version:   "1.0.1",
		Artifacts: artifact.New(),
		Env: map[string]string{
			"FOO": "foo_is_bar",
		},
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			PkgBuilds: []config.PkgBuild{
				{
					License:     "MIT",
					Description: "A run pipe test pkgbuild and FOO={{ .Env.FOO }}",
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
		},
	}
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
			Name:   fmt.Sprintf("%s.tar.gz", a.name),
			Path:   path,
			Goos:   a.goos,
			Goarch: a.goarch,
			Goarm:  a.goarm,
			Type:   artifact.UploadableArchive,
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
	distFile := filepath.Join(folder, "aur/pkgbuilds/foo-bin")

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))

	distBts, err := os.ReadFile(distFile)
	require.NoError(t, err)

	golden.RequireEqual(t, distBts)
	golden.RequireEqual(t, cloneAndGetPKGBUILD(t, url))
}

func TestRunPipeNoBuilds(t *testing.T) {
	ctx := &context.Context{
		TokenType: context.TokenTypeGitHub,
		Config: config.Project{
			ProjectName: "foo",
			PkgBuilds:   []config.PkgBuild{{}},
		},
	}
	client := client.NewMock()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, ErrNoArchivesFound, runAll(ctx, client))
	require.False(t, client.CreatedFile)
}

func TestRunPipeBinaryRelease(t *testing.T) {
	url := makeBareRepo(t)
	key := makeKey(t)
	folder := t.TempDir()
	ctx := &context.Context{
		Git: context.GitInfo{
			CurrentTag: "v1.2.1",
		},
		Version:   "1.2.1",
		Artifacts: artifact.New(),
		Config: config.Project{
			Dist:        folder,
			ProjectName: "foo",
			PkgBuilds: []config.PkgBuild{{
				GitURL:     url,
				PrivateKey: key,
			}},
		},
	}

	path := filepath.Join(folder, "dist/foo_linux_amd64/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "foo_linux_amd64",
		Path:   path,
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID:       "foo",
			artifact.ExtraFormat:   "binary",
			artifact.ExtraBinaries: []string{"foo"},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	client := client.NewMock()
	require.NoError(t, runAll(ctx, client))
	require.NoError(t, publishAll(ctx, client))

	golden.RequireEqual(t, cloneAndGetPKGBUILD(t, url))
}

func TestRunPipeNoUpload(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		PkgBuilds:   []config.PkgBuild{{}},
	})
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
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

	require.NoError(t, Pipe{}.Default(ctx))
	client := client.NewMock()

	assertNoPublish := func(t *testing.T) {
		t.Helper()
		require.NoError(t, runAll(ctx, client))
		testlib.AssertSkipped(t, publishAll(ctx, client))
		require.False(t, client.CreatedFile)
	}
	t.Run("skip upload true", func(t *testing.T) {
		ctx.Config.PkgBuilds[0].SkipUpload = "true"
		ctx.Semver.Prerelease = ""
		assertNoPublish(t)
	})
	t.Run("skip upload auto", func(t *testing.T) {
		ctx.Config.PkgBuilds[0].SkipUpload = "auto"
		ctx.Semver.Prerelease = "beta1"
		assertNoPublish(t)
	})
}

func TestRunEmptyTokenType(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Release:     config.Release{},
		Rigs: []config.GoFish{
			{
				Rig: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	})
	ctx.Git = context.GitInfo{CurrentTag: "v1.0.1"}
	path := filepath.Join(folder, "whatever.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "bin",
		Path:   path,
		Goos:   "darwin",
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
		ctx := &context.Context{
			TokenType: context.TokenTypeGitHub,
			Config: config.Project{
				ProjectName: "myproject",
				PkgBuilds: []config.PkgBuild{
					{},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.PkgBuild{
			Name:                  "myproject-bin",
			Conflicts:             []string{"myproject"},
			Provides:              []string{"myproject"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			SSHCommand:            defaultSSHCommand,
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "goreleaser@carlosbecker.com",
			},
		}, ctx.Config.PkgBuilds[0])
	})

	t.Run("partial", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitHub,
			Config: config.Project{
				ProjectName: "myproject",
				PkgBuilds: []config.PkgBuild{
					{
						Conflicts: []string{"somethingelse"},
					},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.PkgBuild{
			Name:                  "myproject-bin",
			Conflicts:             []string{"somethingelse"},
			Provides:              []string{"myproject"},
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			SSHCommand:            defaultSSHCommand,
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "goreleaser@carlosbecker.com",
			},
		}, ctx.Config.PkgBuilds[0])
	})

	t.Run("name provided", func(t *testing.T) {
		ctx := &context.Context{
			TokenType: context.TokenTypeGitHub,
			Config: config.Project{
				ProjectName: "myproject",
				PkgBuilds: []config.PkgBuild{
					{
						Name: "oops",
					},
				},
			},
		}
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.PkgBuild{
			Name:                  "oops",
			Rel:                   "1",
			CommitMessageTemplate: defaultCommitMsg,
			SSHCommand:            defaultSSHCommand,
			CommitAuthor: config.CommitAuthor{
				Name:  "goreleaserbot",
				Email: "goreleaser@carlosbecker.com",
			},
		}, ctx.Config.PkgBuilds[0])
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			PkgBuilds: []config.PkgBuild{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRunSkipNoName(t *testing.T) {
	ctx := context.New(config.Project{
		PkgBuilds: []config.PkgBuild{{}},
	})

	client := client.NewMock()
	testlib.AssertSkipped(t, runAll(ctx, client))
}

func TestKeyPath(t *testing.T) {
	t.Run("with valid path", func(t *testing.T) {
		path := makeKey(t)
		result, err := keyPath(path)
		require.NoError(t, err)
		require.Equal(t, path, result)
	})
	t.Run("with invalid path", func(t *testing.T) {
		result, err := keyPath("testdata/nope")
		require.EqualError(t, err, `key "testdata/nope" does not exist`)
		require.Equal(t, "", result)
	})
	t.Run("with key", func(t *testing.T) {
		for _, algo := range []keygen.KeyType{keygen.Ed25519, keygen.RSA} {
			t.Run(string(algo), func(t *testing.T) {
				path := makeKey(t, algo)
				bts, err := os.ReadFile(path)
				require.NoError(t, err)

				result, err := keyPath(string(bts))
				require.NoError(t, err)

				resultbts, err := os.ReadFile(result)
				require.NoError(t, err)
				require.Equal(t, string(bts), string(resultbts))
			})
		}
	})
	t.Run("empty", func(t *testing.T) {
		result, err := keyPath("")
		require.EqualError(t, err, `pkgbuild.private_key is empty`)
		require.Equal(t, "", result)
	})
}

func makeBareRepo(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	_, err := git.Run("-C", dir, "init", "--bare", ".")
	require.NoError(tb, err)
	return dir
}

func makeKey(tb testing.TB, algo ...keygen.KeyType) string {
	tb.Helper()

	if len(algo) == 0 {
		algo = append(algo, keygen.Ed25519)
	}
	dir := tb.TempDir()
	k, err := keygen.NewWithWrite(dir, "id", nil, algo[0])
	require.NoError(tb, err)
	return filepath.Join(dir, k.Filename)
}

func cloneAndGetPKGBUILD(tb testing.TB, repo string) []byte {
	tb.Helper()
	dir := tb.TempDir()
	_, err := git.Run("-C", dir, "clone", repo, "repo")
	require.NoError(tb, err)
	bts, err := os.ReadFile(filepath.Join(dir, "repo/PKGBUILD"))
	require.NoError(tb, err)
	return bts
}
