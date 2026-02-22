//go:build integration

package aursources

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestIntegrationFullPipe(t *testing.T) {
	type testcase struct {
		prepare                   func(ctx *context.Context)
		expectedRunError          string
		expectedRunErrorCheck     func(testing.TB, error)
		expectedPublishError      string
		expectedPublishErrorIs    error
		expectedPublishErrorCheck func(testing.TB, error)
	}
	for name, tt := range map[string]testcase{
		"default": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.AURSources[0].Homepage = "https://github.com/goreleaser"
			},
		},
		"custom-dir": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.AURSources[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.AURSources[0].Directory = "foo"
			},
		},
		"with-more-opts": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitHub
				ctx.Config.AURSources[0].Homepage = "https://github.com/goreleaser"
				ctx.Config.AURSources[0].Maintainers = []string{"me"}
				ctx.Config.AURSources[0].Contributors = []string{"me as well"}
				ctx.Config.AURSources[0].Depends = []string{"curl", "bash"}
				ctx.Config.AURSources[0].OptDepends = []string{"wget: stuff", "foo: bar"}
				ctx.Config.AURSources[0].Provides = []string{"git", "svn"}
				ctx.Config.AURSources[0].Conflicts = []string{"libcurl", "cvs", "blah"}
				ctx.Config.AURSources[0].Install = "./testdata/install.sh"
			},
		},
		"default-gitlab": {
			prepare: func(ctx *context.Context) {
				ctx.TokenType = context.TokenTypeGitLab
				ctx.Config.AURSources[0].Homepage = "https://gitlab.com/goreleaser"
			},
		},
		"invalid-name-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].Name = "{{ .Asdsa }"
			},
			expectedRunErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-package-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].Package = "{{ .Asdsa }"
			},
			expectedRunErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-commit-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].CommitMessageTemplate = "{{ .Asdsa }"
			},
			expectedPublishErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-key-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].PrivateKey = "{{ .Asdsa }"
			},
			expectedPublishErrorCheck: testlib.RequireTemplateError,
		},
		"no-key": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].PrivateKey = ""
			},
			expectedPublishError: `private_key is empty`,
		},
		"key-not-found": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].PrivateKey = "testdata/nope"
			},
			expectedPublishErrorIs: os.ErrNotExist,
		},
		"invalid-git-url-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].GitURL = "{{ .Asdsa }"
			},
			expectedPublishErrorCheck: testlib.RequireTemplateError,
		},
		"no-git-url": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].GitURL = ""
			},
			expectedPublishError: `url is empty`,
		},
		"invalid-ssh-cmd-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].GitSSHCommand = "{{ .Asdsa }"
			},
			expectedPublishErrorCheck: testlib.RequireTemplateError,
		},
		"invalid-commit-author-template": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].CommitAuthor.Name = "{{ .Asdsa }"
			},
			expectedPublishErrorCheck: testlib.RequireTemplateError,
		},
		"simple-quote-inside-description": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].Description = "Let's go"
			},
		},
		"double-quote-inside-description": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].Description = `This is a "test"`
			},
		},
		"mixed-quote-inside-description": {
			prepare: func(ctx *context.Context) {
				ctx.Config.AURSources[0].Description = `Let's go, this is a "test"`
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			url := testlib.GitMakeBareRepository(t)
			key := testlib.MakeNewSSHKey(t, "")

			folder := t.TempDir()
			ctx := testctx.WrapWithCfg(t.Context(),
				config.Project{
					Dist:        folder,
					ProjectName: name,
					AURSources: []config.AURSource{
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
				testctx.WithCurrentTag("v1.0.1"),
				testctx.WithSemver(1, 0, 1, ""),
				testctx.WithVersion("1.0.1"),
			)

			tt.prepare(ctx)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "should-be-ignored.tar.gz",
				Path:    "doesnt matter",
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v3",
				Type:    artifact.UploadableArchive,
				Extra: map[string]any{
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
				Extra: map[string]any{
					artifact.ExtraID:       "bar",
					artifact.ExtraFormat:   "tar.gz",
					artifact.ExtraBinaries: []string{"bar"},
				},
			})
			path := filepath.Join(folder, "sources.tar.gz")

			ctx.Artifacts.Add(&artifact.Artifact{
				Name: "sources.tar.gz",
				Path: path,
				Type: artifact.UploadableSourceArchive,
				Extra: map[string]any{
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
			if tt.expectedRunErrorCheck != nil {
				tt.expectedRunErrorCheck(t, runAll(ctx, client))
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

			if tt.expectedPublishErrorCheck != nil {
				tt.expectedPublishErrorCheck(t, Pipe{}.Publish(ctx))
				return
			}

			require.NoError(t, Pipe{}.Publish(ctx))

			requireEqualRepoFiles(t, folder, ctx.Config.AURSources[0].Directory, name, url)
		})
	}
}

func TestIntegrationRunPipe(t *testing.T) {
	url := testlib.GitMakeBareRepository(t)
	key := testlib.MakeNewSSHKey(t, "")

	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURSources: []config.AURSource{
				{
					License:     "MIT",
					Description: "A run pipe test aur and FOO={{ .Env.FOO }}",
					Homepage:    "https://github.com/goreleaser",
					IDs:         []string{"foo"},
					GitURL:      url,
					PrivateKey:  key,
					Install:     "./testdata/install.sh",
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
		name string
	}{
		{
			name: "source",
		},
	} {
		path := filepath.Join(folder, fmt.Sprintf("%s.tar.gz", a.name))
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:    fmt.Sprintf("%s.tar.gz", a.name),
			Path:    path,
			Goamd64: "v1",
			Type:    artifact.UploadableSourceArchive,
			Extra: map[string]any{
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

	requireEqualRepoFilesMap(t, ".", url, map[string]string{
		"PKGBUILD":    filepath.Join(folder, "aur", "foo.pkgbuild"),
		".SRCINFO":    filepath.Join(folder, "aur", "foo.srcinfo"),
		"foo.install": "./testdata/install.sh",
	})
}

func TestIntegrationRunPipeMultipleConfigurations(t *testing.T) {
	url := testlib.GitMakeBareRepository(t)
	key := testlib.MakeNewSSHKey(t, "")

	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURSources: []config.AURSource{
				{
					Disable: `{{printf "true"}}`,
				},
				{
					Name:        "foo",
					IDs:         []string{"foo"},
					PrivateKey:  key,
					License:     "MIT",
					GitURL:      url,
					Description: "The foo aur",
					Directory:   "foo",
				},
				{
					Name:        "bar",
					IDs:         []string{"bar"},
					PrivateKey:  key,
					License:     "MIT",
					GitURL:      url,
					Description: "The bar aur",
					Directory:   "bar",
				},
			},
		},
		testctx.WithCurrentTag("v1.0.1-foo"),
		testctx.WithSemver(1, 0, 1, "foo"),
		testctx.WithVersion("1.0.1-foo"),
	)

	path := filepath.Join(folder, "source.tar.gz")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "source_bin.tar.gz",
		Path:    path,
		Goamd64: "v1",
		Type:    artifact.UploadableSourceArchive,
		Extra: map[string]any{
			artifact.ExtraID:       "bar",
			artifact.ExtraFormat:   "tar.gz",
			artifact.ExtraBinaries: []string{"bar"},
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "source.tar.gz",
		Path:    path,
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableSourceArchive,
		Extra: map[string]any{
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
	require.True(t, pipe.IsSkip(runAll(ctx, client)), "should partially skip")
	require.NoError(t, Pipe{}.Publish(ctx))

	dir := t.TempDir()
	_, err = git.Run(t.Context(), "-C", dir, "clone", url, "repo")
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(dir, "repo", "foo", ".SRCINFO"))
	require.FileExists(t, filepath.Join(dir, "repo", "foo", "PKGBUILD"))
	require.FileExists(t, filepath.Join(dir, "repo", "bar", ".SRCINFO"))
	require.FileExists(t, filepath.Join(dir, "repo", "bar", "PKGBUILD"))
}

func TestIntegrationRunPipeWrappedInDirectory(t *testing.T) {
	url := testlib.GitMakeBareRepository(t)
	key := testlib.MakeNewSSHKey(t, "")
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURSources: []config.AURSource{{
				GitURL:     url,
				PrivateKey: key,
			}},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
		testctx.WithSemver(1, 2, 1, ""),
	)

	path := filepath.Join(folder, "dist/sources/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "source.tar.gz",
		Path:    path,
		Goamd64: "v1",
		Type:    artifact.UploadableSourceArchive,
		Extra: map[string]any{
			artifact.ExtraID:        "foo",
			artifact.ExtraFormat:    "tar.gz",
			artifact.ExtraBinaries:  []string{"foo"},
			artifact.ExtraWrappedIn: "foo",
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

	requireEqualRepoFiles(t, folder, ".", "foo", url)
}

func TestIntegrationRunPipeBinaryRelease(t *testing.T) {
	url := testlib.GitMakeBareRepository(t)
	key := testlib.MakeNewSSHKey(t, "")
	folder := t.TempDir()
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			AURSources: []config.AURSource{{
				GitURL:     url,
				PrivateKey: key,
			}},
		},
		testctx.WithVersion("1.2.1"),
		testctx.WithCurrentTag("v1.2.1"),
		testctx.WithSemver(1, 2, 1, ""),
	)

	path := filepath.Join(folder, "dist/sources/foo")
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "sources",
		Path:    path,
		Goamd64: "v1",
		Type:    artifact.UploadableSourceArchive,
		Extra: map[string]any{
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

	requireEqualRepoFiles(t, folder, ".", "foo", url)
}

func requireEqualRepoFilesMap(tb testing.TB, repoDir, url string, files map[string]string) {
	tb.Helper()
	dir := tb.TempDir()
	_, err := git.Run(tb.Context(), "-C", dir, "clone", url, "repo")
	require.NoError(tb, err)

	for reponame, distpath := range files {
		bts, err := os.ReadFile(distpath)
		require.NoError(tb, err)
		ext := filepath.Ext(distpath)
		golden.RequireEqualExt(tb, bts, ext)

		bts, err = os.ReadFile(filepath.Join(dir, "repo", repoDir, reponame))
		require.NoError(tb, err)
		golden.RequireEqualExt(tb, bts, ext)
	}
}

func requireEqualRepoFiles(tb testing.TB, distDir, repoDir, name, url string) {
	tb.Helper()
	requireEqualRepoFilesMap(tb, repoDir, url, map[string]string{
		"PKGBUILD": filepath.Join(distDir, "aur", name+".pkgbuild"),
		".SRCINFO": filepath.Join(distDir, "aur", name+".srcinfo"),
	})
}
