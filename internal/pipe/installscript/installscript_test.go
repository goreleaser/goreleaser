package installscript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	pipepkg "github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestContinueOnError(t *testing.T) {
	require.True(t, Pipe{}.ContinueOnError())
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
	})
	t.Run("skip flag", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			InstallScript: config.InstallScript{
				Repository: config.RepoRef{Name: "repo"},
			},
		}, testctx.Skip(skips.InstallScript))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("configured", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			InstallScript: config.InstallScript{
				Repository: config.RepoRef{Name: "repo"},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "myapp",
		InstallScript: config.InstallScript{
			Repository: config.RepoRef{
				Name: "repo",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	cfg := ctx.Config.InstallScript
	require.Equal(t, "www/static", cfg.Directory)
	require.Equal(t, "/usr/local/bin", cfg.InstallTo.Linux)
	require.Equal(t, "/usr/local/bin", cfg.InstallTo.Darwin)
	require.Equal(t, "$env:ProgramFiles\\{{ .ProjectName }}", cfg.InstallTo.Windows)
	require.Equal(t, "v1", cfg.Goamd64)
	require.NotEmpty(t, cfg.CommitMessageTemplate)
	require.NotEmpty(t, cfg.CommitAuthor.Name)
	require.NotEmpty(t, cfg.CommitAuthor.Email)
}

func TestRunEnable(t *testing.T) {
	t.Run("default enabled when unset", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			InstallScript: config.InstallScript{
				Repository: config.RepoRef{Name: "repo"},
			},
		})
		require.EqualError(t, Pipe{}.Run(ctx), errNoScriptDirectory.Error())
	})

	t.Run("explicit false skips", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			InstallScript: config.InstallScript{
				Enable:     "false",
				Repository: config.RepoRef{Name: "repo"},
			},
		})
		err := Pipe{}.Run(ctx)
		require.EqualError(t, err, "install_script.enable evaluates to false")
		require.True(t, pipepkg.IsSkip(err))
	})

	t.Run("templated false skips", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(
			t.Context(),
			config.Project{
				InstallScript: config.InstallScript{
					Enable:     "{{ .Env.SKIP_INSTALL_SCRIPT }}",
					Repository: config.RepoRef{Name: "repo"},
				},
			},
			testctx.WithEnv(map[string]string{"SKIP_INSTALL_SCRIPT": "false"}),
		)
		err := Pipe{}.Run(ctx)
		require.EqualError(t, err, "install_script.enable evaluates to false")
		require.True(t, pipepkg.IsSkip(err))
	})
}

func TestPublishEnable(t *testing.T) {
	t.Run("explicit false skips", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			InstallScript: config.InstallScript{
				Enable: "false",
				Repository: config.RepoRef{
					Owner: "acme",
					Name:  "scripts",
				},
			},
		})
		err := Pipe{}.Publish(ctx)
		require.EqualError(t, err, "install_script.enable evaluates to false")
		require.True(t, pipepkg.IsSkip(err))
	})

	t.Run("default enabled when unset", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(
			t.Context(),
			config.Project{
				InstallScript: config.InstallScript{
					Repository: config.RepoRef{
						Owner: "acme",
						Name:  "scripts",
					},
				},
			},
			testctx.GitHubTokenType,
		)
		err := Pipe{}.Publish(ctx)
		require.EqualError(t, err, "install_script artifacts not found")
		require.True(t, pipepkg.IsSkip(err))
	})
}

func TestRunAndPublish(t *testing.T) {
	dir := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        dir,
			ProjectName: "foo",
			InstallScript: config.InstallScript{
				Directory: "www/custom/scripts",
				Repository: config.RepoRef{
					Owner: "acme",
					Name:  "scripts",
				},
				InstallTo: config.InstallScriptInstallTo{
					Linux:   "/opt/foo/linux/bin",
					Darwin:  "/opt/foo/darwin/bin",
					Windows: "$env:ProgramFiles\\FooCustom",
				},
				MessageBefore: "before",
				MessageAfter:  "after",
			},
		},
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithVersion("1.2.3"),
	)
	require.NoError(t, Pipe{}.Default(ctx))

	bin := filepath.Join(dir, "foo")
	winBin := filepath.Join(dir, "foo.exe")
	require.NoError(t, os.WriteFile(bin, []byte("bin"), 0o644))
	require.NoError(t, os.WriteFile(winBin, []byte("bin"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.UploadableBinary,
		Name:    "foo_1.2.3_linux_amd64",
		Path:    bin,
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Extra:   map[string]any{artifact.ExtraID: "default", artifact.ExtraBinary: "foo"},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.2.3_darwin_arm64.tar.gz",
		Path:   filepath.Join(dir, "darwin.tar.gz"),
		Goos:   "darwin",
		Goarch: "arm64",
		Extra:  map[string]any{artifact.ExtraID: "default", artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.2.3_windows_arm64.zip",
		Path:   filepath.Join(dir, "windows.zip"),
		Goos:   "windows",
		Goarch: "arm64",
		Extra:  map[string]any{artifact.ExtraID: "default", artifact.ExtraFormat: "zip", artifact.ExtraBinaries: []string{"foo.exe"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.UploadableBinary,
		Name:    "foo_1.2.3_windows_amd64.exe",
		Path:    winBin,
		Goos:    "windows",
		Goarch:  "amd64",
		Goamd64: "v1",
		Extra:   map[string]any{artifact.ExtraID: "default", artifact.ExtraBinary: "foo.exe"},
	})

	require.NoError(t, Pipe{}.Run(ctx))
	scripts := ctx.Artifacts.Filter(artifact.ByType(artifact.InstallScript)).List()
	require.Len(t, scripts, 2)

	var unixScript *artifact.Artifact
	var windowsScript *artifact.Artifact
	for _, script := range scripts {
		switch artifact.MustExtra[string](*script, installScriptKindExtra) {
		case scriptKindUnix:
			unixScript = script
		case scriptKindWindows:
			windowsScript = script
		}
	}
	require.NotNil(t, unixScript)
	require.NotNil(t, windowsScript)
	require.Equal(t, "install.sh", unixScript.Name)
	require.Equal(t, "install.ps1", windowsScript.Name)
	require.Equal(t, "www/custom/scripts/install.sh", artifact.MustExtra[string](*unixScript, installScriptPathExtra))
	require.Equal(t, "www/custom/scripts/install.ps1", artifact.MustExtra[string](*windowsScript, installScriptPathExtra))

	unixContent, err := os.ReadFile(unixScript.Path)
	require.NoError(t, err)
	golden.RequireEqualExt(t, unixContent, ".sh")

	windowsContent, err := os.ReadFile(windowsScript.Path)
	require.NoError(t, err)
	golden.RequireEqualExt(t, windowsContent, ".ps1")

	mock := client.NewMock()
	require.NoError(t, publishAll(ctx, ctx.Config.InstallScript, mock))
	require.True(t, mock.CreatedFile)
	require.Len(t, mock.Messages, 2)
	require.Contains(t, mock.Messages[0], "version v1.2.3")
}

func TestPublishCreatesFilesWhenContentIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        dir,
			ProjectName: "foo",
			InstallScript: config.InstallScript{
				Repository: config.RepoRef{
					Owner: "acme",
					Name:  "scripts",
				},
			},
		},
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithVersion("1.0.0"),
	)
	require.NoError(t, Pipe{}.Default(ctx))

	path := filepath.Join(dir, "install.sh")
	require.NoError(t, os.WriteFile(path, []byte("same"), 0o644))
	ctx.Artifacts.Add(&artifact.Artifact{
		Type: artifact.InstallScript,
		Name: "install.sh",
		Path: path,
		Extra: map[string]any{
			artifact.ExtraID:       installScriptIDUnix,
			installScriptPathExtra: "www/static/install.sh",
		},
	})

	mock := client.NewMock()
	require.NoError(t, publishAll(ctx, ctx.Config.InstallScript, mock))
	require.True(t, mock.CreatedFile)
	require.Len(t, mock.Messages, 1)
}

func TestAssetSelectionPrecedence(t *testing.T) {
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			ProjectName: "foo",
		},
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithVersion("1.0.0"),
	)
	cfg := config.InstallScript{
		URLTemplate: "https://example.com/{{ .ArtifactName }}",
		Goamd64:     "v1",
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.0.0_linux_amd64.zip",
		Goos:   "linux",
		Goarch: "amd64",
		Extra:  map[string]any{artifact.ExtraFormat: "zip", artifact.ExtraBinaries: []string{"foo"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.0.0_linux_amd64.tar.gz",
		Goos:   "linux",
		Goarch: "amd64",
		Extra:  map[string]any{artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.UploadableBinary,
		Name:    "foo_1.0.0_linux_amd64",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Extra:   map[string]any{artifact.ExtraBinary: "foo"},
	})

	assets, err := selectAssets(ctx, cfg, "linux")
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "binary", assets[0].Kind)
}

func TestRunIncludesDarwinAllFallbackSelection(t *testing.T) {
	dir := t.TempDir()
	ctx := testctx.WrapWithCfg(
		t.Context(),
		config.Project{
			Dist:        dir,
			ProjectName: "foo",
			InstallScript: config.InstallScript{
				Directory: "www/static",
				Repository: config.RepoRef{
					Owner: "acme",
					Name:  "scripts",
				},
				InstallTo: config.InstallScriptInstallTo{
					Linux:   "/usr/local/bin",
					Darwin:  "/usr/local/bin",
					Windows: "$env:ProgramFiles\\Foo",
				},
			},
		},
		testctx.GitHubTokenType,
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithVersion("1.2.3"),
	)
	require.NoError(t, Pipe{}.Default(ctx))

	linuxBin := filepath.Join(dir, "foo")
	winBin := filepath.Join(dir, "foo.exe")
	require.NoError(t, os.WriteFile(linuxBin, []byte("bin"), 0o644))
	require.NoError(t, os.WriteFile(winBin, []byte("bin"), 0o644))

	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.UploadableBinary,
		Name:    "foo_1.2.3_linux_amd64",
		Path:    linuxBin,
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Extra:   map[string]any{artifact.ExtraID: "default", artifact.ExtraBinary: "foo"},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.2.3_darwin_all.tar.gz",
		Path:   filepath.Join(dir, "darwin_all.tar.gz"),
		Goos:   "darwin",
		Goarch: "all",
		Extra:  map[string]any{artifact.ExtraID: "default", artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.2.3_darwin_arm64.tar.gz",
		Path:   filepath.Join(dir, "darwin_arm64.tar.gz"),
		Goos:   "darwin",
		Goarch: "arm64",
		Extra:  map[string]any{artifact.ExtraID: "default", artifact.ExtraFormat: "tar.gz", artifact.ExtraBinaries: []string{"foo"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:   artifact.UploadableArchive,
		Name:   "foo_1.2.3_windows_all.zip",
		Path:   filepath.Join(dir, "windows_all.zip"),
		Goos:   "windows",
		Goarch: "all",
		Extra:  map[string]any{artifact.ExtraID: "default", artifact.ExtraFormat: "zip", artifact.ExtraBinaries: []string{"foo.exe"}},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.UploadableBinary,
		Name:    "foo_1.2.3_windows_amd64.exe",
		Path:    winBin,
		Goos:    "windows",
		Goarch:  "amd64",
		Goamd64: "v1",
		Extra:   map[string]any{artifact.ExtraID: "default", artifact.ExtraBinary: "foo.exe"},
	})

	require.NoError(t, Pipe{}.Run(ctx))
	scripts := ctx.Artifacts.Filter(artifact.ByType(artifact.InstallScript)).List()
	require.Len(t, scripts, 2)

	var unixScript *artifact.Artifact
	var windowsScript *artifact.Artifact
	for _, script := range scripts {
		switch artifact.MustExtra[string](*script, installScriptKindExtra) {
		case scriptKindUnix:
			unixScript = script
		case scriptKindWindows:
			windowsScript = script
		}
	}
	require.NotNil(t, unixScript)
	require.NotNil(t, windowsScript)

	content, err := os.ReadFile(unixScript.Path)
	require.NoError(t, err)
	golden.RequireEqualExt(t, content, ".sh")

	winContent, err := os.ReadFile(windowsScript.Path)
	require.NoError(t, err)
	golden.RequireEqualExt(t, winContent, ".ps1")
}
